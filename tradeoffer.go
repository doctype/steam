package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	TradeStateNone = iota
	TradeStateInvalid
	TradeStateActive
	TradeStateAccepted
	TradeStateCountered
	TradeStateExpired
	TradeStateCanceled
	TradeStateDeclined
	TradeStateInvalidItems
	TradeStateCreatedNeedsConfirmation
	TradeStatePendingConfirmation
	TradeStateEmailPending
	TradeStateCanceledByTwoFactor
	TradeStateCanceledConfirmation
	TradeStateEmailCanceled
	TradeStateInEscrow
)

const (
	TradeConfirmationNone = iota
	TradeConfirmationEmail
	TradeConfirmationMobileApp
	TradeConfirmationMobile
)

const (
	TradeFilterNone           = iota
	TradeFilterSentOffers     = 1 << 0
	TradeFilterRecvOffers     = 1 << 1
	TradeFilterActiveOnly     = 1 << 3
	TradeFilterHistoricalOnly = 1 << 4
)

type EconItem struct {
	AssetID    uint64 `json:"assetid,string,omitempty"`
	InstanceID uint64 `json:"instanceid,string,omitempty"`
	ClassID    uint64 `json:"classid,string,omitempty"`
	AppID      uint64 `json:"appid,string"`
	ContextID  uint64 `json:"contextid,string"`
	Amount     uint64 `json:"amount,string"`
	Missing    bool   `json:"missing,omitempty"`
}

type TradeOffer struct {
	ID                 uint64      `json:"tradeofferid,string"`
	Partner            uint32      `json:"accountid_other"`
	ReceiptID          uint64      `json:"tradeid,string"`
	ReceiveItems       []*EconItem `json:"items_to_receive"`
	SendItems          []*EconItem `json:"items_to_give"`
	Message            string      `json:"message"`
	State              uint8       `json:"trade_offer_state"`
	ConfirmationMethod uint8       `json:"confirmation_method"`
	Created            uint64      `json:"time_created"`
	Updated            uint64      `json:"time_updated"`
	Expires            uint64      `json:"expiration_time"`
	EscrowEndDate      uint64      `json:"escrow_end_date"`
	RealTime           bool        `json:"from_real_time_trade"`
	IsOurOffer         bool        `json:"is_our_offer"`
}

type TradeOfferResponse struct {
	Offer          *TradeOffer   `json:"offer"`
	SentOffers     []*TradeOffer `json:"trade_offers_sent"`
	ReceivedOffers []*TradeOffer `json:"trade_offers_received"`
}

type APIResponse struct {
	Inner TradeOfferResponse `json:"response"`
}

func (community *Community) GetTradeOffer(id uint64) (*TradeOffer, error) {
	values := url.Values{}
	values.Add("key", community.apiKey)
	values.Add("tradeofferid", strconv.FormatUint(id, 10))

	body, err := community.MakeAPICall(http.MethodGet, "GetTradeOffer", &values)
	if err != nil {
		return nil, err
	}

	var response APIResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, err
	}

	return response.Inner.Offer, nil
}

func test_bit(bits uint32, bit uint32) bool {
	return (bits & bit) == bit
}

func (community *Community) GetTradeOffers(filter uint32, timeCutOff time.Time) (sentOffers []*TradeOffer, recvOffers []*TradeOffer, err error) {
	values := url.Values{}
	values.Add("key", community.apiKey)

	if test_bit(filter, TradeFilterSentOffers) {
		values.Add("get_sent_offers", "1")
	}

	if test_bit(filter, TradeFilterRecvOffers) {
		values.Add("get_received_offers", "1")
	}

	if test_bit(filter, TradeFilterActiveOnly) {
		values.Add("active_only", "1")
	}

	if test_bit(filter, TradeFilterHistoricalOnly) {
		values.Add("historical_only", "1")
		values.Add("time_historical_cutoff", strconv.FormatInt(timeCutOff.Unix(), 10))
	}

	body, err := community.MakeAPICall(http.MethodGet, "GetTradeOffers", &values)
	if err != nil {
		return nil, nil, err
	}

	var response APIResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return nil, nil, err
	}

	return response.Inner.SentOffers, response.Inner.ReceivedOffers, nil
}

func (community *Community) SendTradeOffer(offer *TradeOffer, sid SteamID, token string) error {
	content := map[string]interface{}{
		"newversion": true,
		"version":    3,
		"me": map[string]interface{}{
			"assets":   offer.SendItems,
			"currency": make([]struct{}, 0),
			"ready":    false,
		},
		"them": map[string]interface{}{
			"assets":   offer.ReceiveItems,
			"currency": make([]struct{}, 0),
			"ready":    false,
		},
	}

	contentJson, err := json.Marshal(content)
	if err != nil {
		return err
	}

	accessToken := map[string]string{
		"trade_offer_access_token": token,
	}
	params, err := json.Marshal(accessToken)
	if err != nil {
		return err
	}

	body := url.Values{
		"sessionid":                 {community.sessionID},
		"serverid":                  {"1"},
		"partner":                   {sid.ToString()},
		"tradeoffermessage":         {offer.Message},
		"json_tradeoffer":           {string(contentJson)},
		"trade_offer_create_params": {string(params)},
	}

	req, err := http.NewRequest(http.MethodPost, "https://steamcommunity.com/tradeoffer/new/send", strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	req.Header.Add("Referer", fmt.Sprintf("https://steamcommunity.com/tradeoffer/new/?partner=%d&token=%s", sid.GetAccountID(), token))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := community.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	type Response struct {
		ErrorMessage               string `json:"strError"`
		ID                         uint64 `json:"tradeofferid,string"`
		MobileConfirmationRequired bool   `json:"needs_mobile_confirmation"`
		EmailConfirmationRequired  bool   `json:"needs_email_confirmation"`
		EmailDomain                string `json:"email_domain"`
	}

	var j Response
	if err = json.NewDecoder(resp.Body).Decode(&j); err != nil {
		return err
	}

	if len(j.ErrorMessage) != 0 {
		return errors.New(j.ErrorMessage)
	}

	if j.ID != 0 {
		offer.ID = j.ID

		// Just test mobile confirmation, email is deprecated
		if j.MobileConfirmationRequired {
			offer.ConfirmationMethod = TradeConfirmationMobileApp
			offer.State = TradeStateCreatedNeedsConfirmation
		} else {
			// set state to active
			offer.State = TradeStateActive
		}

		return nil
	}

	return errors.New("No OfferID included")
}

func (community *Community) GetTradeReceivedItems(receiptID uint32) (items []*EconItem, err error) {
	return items, err
}

func (community *Community) DeclineTradeOffer(id uint64) error {
	values := url.Values{}
	values.Add("key", community.apiKey)
	values.Add("tradeofferid", strconv.FormatUint(id, 10))

	body, err := community.MakeAPICall(http.MethodPost, "DeclineTradeOffer", &values)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func (community *Community) CancelTradeOffer(id uint64) error {
	values := url.Values{}
	values.Add("key", community.apiKey)
	values.Add("tradeofferid", strconv.FormatUint(id, 10))

	body, err := community.MakeAPICall(http.MethodPost, "CancelTradeOffer", &values)
	if err != nil {
		return err
	}

	fmt.Println(string(body))
	return nil
}

func (community *Community) AcceptTradeOffer(id uint64) error {
	return nil
}

func (offer *TradeOffer) Accept() error {
	return nil
}

func (offer *TradeOffer) Cancel() error {
	return nil
}

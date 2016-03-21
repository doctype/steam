/*
   Steam Library For Go
   Copyright (C) 2016 Ahmed Samy <f.fallen45@gmail.com>

   This library is free software; you can redistribute it and/or
   modify it under the terms of the GNU Lesser General Public
   License as published by the Free Software Foundation; either
   version 2.1 of the License, or (at your option) any later version.

   This library is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
   Lesser General Public License for more details.

   You should have received a copy of the GNU Lesser General Public
   License along with this library; if not, write to the Free Software
   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
*/
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
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

var (
	// receiptExp matches JSON in the following form:
	//	oItem = {"id":"...",...}; (Javascript code)
	receiptExp = regexp.MustCompile("oItem =\\s(.+?});")
	apiCallURL = "https://api.steampowered.com/IEconService/"

	ErrReceiptMatch      = errors.New("unable to match items in trade receipt")
	ErrCannotCancelTrade = errors.New("unable to cancel/decline specified trade")
)

// Due to the JSON being string, etc... we cannot re-use item
// Also, "assetid" is included as "id" not as assetid.
type ReceiptItem struct {
	AssetID        uint64 `json:"id,string,omitempty"`
	InstanceID     uint64 `json:"instanceid,string,omitempty"`
	ClassID        uint64 `json:"classid,string,omitempty"`
	AppID          uint32 `json:"appid"`     // This!
	ContextID      uint16 `json:"contextid"` // Ditto
	Name           string `json:"name"`
	MarketHashName string `json:"market_hash_name"`
}

type EconItem struct {
	AssetID    uint64 `json:"assetid,string,omitempty"`
	InstanceID uint64 `json:"instanceid,string,omitempty"`
	ClassID    uint64 `json:"classid,string,omitempty"`
	AppID      uint32 `json:"appid,string"`
	ContextID  uint16 `json:"contextid,string"`
	Amount     uint16 `json:"amount,string"`
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
	Success        bool          `json:"success"`               // {Decline,Cancel}TradeOffer
	Offer          *TradeOffer   `json:"offer"`                 // GetTradeOffer
	SentOffers     []*TradeOffer `json:"trade_offers_sent"`     // GetTradeOffers
	ReceivedOffers []*TradeOffer `json:"trade_offers_received"` // GetTradeOffers
}

type APIResponse struct {
	Inner TradeOfferResponse `json:"response"`
}

func (community *Community) GetTradeOffer(id uint64) (*TradeOffer, error) {
	resp, err := community.client.Get(fmt.Sprintf("%s/GetTradeOffer/v1/?key=%s&Tradeofferid=%d", apiCallURL, community.apiKey, id))
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	var response APIResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner.Offer, nil
}

func testBit(bits uint32, bit uint32) bool {
	return (bits & bit) == bit
}

func (community *Community) GetTradeOffers(filter uint32, timeCutOff time.Time) ([]*TradeOffer, []*TradeOffer, error) {
	values := "key=" + community.apiKey
	if testBit(filter, TradeFilterSentOffers) {
		values += "&get_sent_offers=1"
	}

	if testBit(filter, TradeFilterRecvOffers) {
		values += "&get_received_offers=1"
	}

	if testBit(filter, TradeFilterActiveOnly) {
		values += "&active_only=1"
	}

	if testBit(filter, TradeFilterHistoricalOnly) {
		values += "&historical_only=1&time_historical_cutoff=" + strconv.FormatInt(timeCutOff.Unix(), 10)
	}

	resp, err := community.client.Get(fmt.Sprintf("%s/GetTradeOffers/v1/?%s", apiCallURL, values))
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, nil, err
	}

	var response APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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

	contentJSON, err := json.Marshal(content)
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
		"json_tradeoffer":           {string(contentJSON)},
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

	if j.ID == 0 {
		return errors.New("no OfferID included")
	}

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

func (community *Community) GetTradeReceivedItems(receiptID uint64) ([]*ReceiptItem, error) {
	resp, err := community.client.Get(fmt.Sprintf("https://steamcommunity.com/trade/%d/receipt", receiptID))
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := receiptExp.FindAllSubmatch(body, -1)
	if m == nil {
		return nil, ErrReceiptMatch
	}

	items := []*ReceiptItem{}
	for k := range m {
		var item ReceiptItem
		if err = json.Unmarshal(m[k][1], &item); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

func (community *Community) DeclineTradeOffer(id uint64) error {
	values := url.Values{}
	values.Set("key", community.apiKey)
	values.Set("tradeofferid", strconv.FormatUint(id, 10))

	resp, err := community.client.PostForm(apiCallURL+"/DeclineTradeOffer/v1/", values)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var response APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Inner.Success {
		return ErrCannotCancelTrade
	}

	return nil
}

func (community *Community) CancelTradeOffer(id uint64) error {
	values := url.Values{}
	values.Set("key", community.apiKey)
	values.Set("tradeofferid", strconv.FormatUint(id, 10))

	resp, err := community.client.PostForm(apiCallURL+"/CancelTradeOffer/v1/", values)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var response APIResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Inner.Success {
		return ErrCannotCancelTrade
	}

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

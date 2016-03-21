package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
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
	AssetID    string `json:"assetid"`
	InstanceID string `json:"instanceid"`
	ClassID    string `json:"classid"`
	AppID      string `json:"appid"`
	ContextID  string `json:"contextid"`
	Amount     string `json:"amount"`
	Missing    bool   `json:"missing"`
}

type TradeOffer struct {
	ID                 string      `json:"tradeofferid"`
	Partner            uint32      `json:"accountid_other"`
	ReceiptID          string      `json:"tradeid"`
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

func (community *Community) SendTradeOffer(offer *TradeOffer) error {
	return nil
}

func (community *Community) GetTradeReceivedItems(receiptID uint32) (items []*EconItem, err error) {
	return items, err
}

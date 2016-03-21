package main

import (
	"encoding/json"
	"fmt"
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
	Offer *TradeOffer `json:"offer"`
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

	fmt.Println(response)
	return response.Inner.Offer, nil
}

func (community *Community) GetTradeOffers(filter uint32, timeCutOff time.Time) (offers []*TradeOffer, err error) {
	return
}

func (community *Community) SendTradeOffer(offer *TradeOffer) error {
	return nil
}

func (community *Community) GetTradeReceivedItems(receiptID uint32) (items []*EconItem, err error) {
	return items, err
}

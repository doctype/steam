package steam

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
	TradeStateCanceledByTwoFactor
	TradeStateInEscrow
)

const (
	TradeConfirmationNone = iota
	TradeConfirmationEmail
	TradeConfirmationMobileApp
	TradeConfirmationMobile
)

const (
	TradeFilterNone             = iota
	TradeFilterSentOffers       = 1 << 0
	TradeFilterRecvOffers       = 1 << 1
	TradeFilterActiveOnly       = 1 << 3
	TradeFilterHistoricalOnly   = 1 << 4
	TradeFilterItemDescriptions = 1 << 5
)

var (
	// receiptExp matches JSON in the following form:
	//	oItem = {"id":"...",...}; (Javascript code)
	receiptExp    = regexp.MustCompile("oItem =\\s(.+?});")
	myEscrowExp   = regexp.MustCompile("var g_daysMyEscrow = (\\d+);")
	themEscrowExp = regexp.MustCompile("var g_daysTheirEscrow = (\\d+);")
	errorMsgExp   = regexp.MustCompile("<div id=\"error_msg\">\\s*([^<]+)\\s*</div>")
	offerInfoExp  = regexp.MustCompile("token=([a-zA-Z0-9-_]+)")

	apiGetTradeOffer     = "https://api.steampowered.com/IEconService/GetTradeOffer/v1/?"
	apiGetTradeOffers    = "https://api.steampowered.com/IEconService/GetTradeOffers/v1/?"
	apiDeclineTradeOffer = "https://api.steampowered.com/IEconService/DeclineTradeOffer/v1/"
	apiCancelTradeOffer  = "https://api.steampowered.com/IEconService/CancelTradeOffer/v1/"

	ErrReceiptMatch        = errors.New("unable to match items in trade receipt")
	ErrCannotAcceptActive  = errors.New("unable to accept a non-active trade")
	ErrCannotFindOfferInfo = errors.New("unable to match data from trade offer url")
)

type EconItem struct {
	AssetID    uint64 `json:"assetid,string,omitempty"`
	InstanceID uint64 `json:"instanceid,string,omitempty"`
	ClassID    uint64 `json:"classid,string,omitempty"`
	AppID      uint32 `json:"appid"`
	ContextID  uint64 `json:"contextid,string"`
	Amount     uint16 `json:"amount,string"`
	Missing    bool   `json:"missing,omitempty"`
}

type EconDesc struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Color string `json:"color"`
}

type EconTag struct {
	InternalName string `json:"internal_name"`
	Name         string `json:"name"`
	Category     string `json:"category"`
	CategoryName string `json:"category_name"`
}

type EconAction struct {
	Link string `json:"link"`
	Name string `json:"name"`
}

type EconItemDesc struct {
	ClassID         uint64        `json:"classid,string"`    // for matching with EconItem
	InstanceID      uint64        `json:"instanceid,string"` // for matching with EconItem
	Tradable        bool          `json:"tradable"`
	BackgroundColor string        `json:"background_color"`
	IconURL         string        `json:"icon_url"`
	IconLargeURL    string        `json:"icon_url_large"`
	IconDragURL     string        `json:"icon_drag_url"`
	Name            string        `json:"name"`
	NameColor       string        `json:"name_color"`
	MarketName      string        `json:"market_name"`
	MarketHashName  string        `json:"market_hash_name"`
	Comodity        bool          `json:"comodity"`
	Actions         []*EconAction `json:"actions"`
	Tags            []*EconTag    `json:"tags"`
	Descriptions    []*EconDesc   `json:"descriptions"`
}

type TradeOffer struct {
	ID                 uint64      `json:"tradeofferid,string"`
	Partner            uint32      `json:"accountid_other"`
	ReceiptID          uint64      `json:"tradeid,string"`
	RecvItems          []*EconItem `json:"items_to_receive"`
	SendItems          []*EconItem `json:"items_to_give"`
	Message            string      `json:"message"`
	State              uint8       `json:"trade_offer_state"`
	ConfirmationMethod uint8       `json:"confirmation_method"`
	Created            int64       `json:"time_created"`
	Updated            int64       `json:"time_updated"`
	Expires            int64       `json:"expiration_time"`
	EscrowEndDate      int64       `json:"escrow_end_date"`
	RealTime           bool        `json:"from_real_time_trade"`
	IsOurOffer         bool        `json:"is_our_offer"`
}

type TradeOfferResponse struct {
	Offer          *TradeOffer     `json:"offer"`                 // GetTradeOffer
	SentOffers     []*TradeOffer   `json:"trade_offers_sent"`     // GetTradeOffers
	ReceivedOffers []*TradeOffer   `json:"trade_offers_received"` // GetTradeOffers
	Descriptions   []*EconItemDesc `json:"descriptions"`          // GetTradeOffers
}

type APIResponse struct {
	Inner *TradeOfferResponse `json:"response"`
}

func (session *Session) GetTradeOffer(id uint64) (*TradeOffer, error) {
	resp, err := session.client.Get(apiGetTradeOffer + url.Values{
		"key":          {session.apiKey},
		"tradeofferid": {strconv.FormatUint(id, 10)},
	}.Encode())
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

func (session *Session) GetTradeOffers(filter uint32, timeCutOff time.Time) (*TradeOfferResponse, error) {
	params := url.Values{
		"key": {session.apiKey},
	}
	if testBit(filter, TradeFilterSentOffers) {
		params.Set("get_sent_offers", "1")
	}

	if testBit(filter, TradeFilterRecvOffers) {
		params.Set("get_received_offers", "1")
	}

	if testBit(filter, TradeFilterActiveOnly) {
		params.Set("active_only", "1")
	}

	if testBit(filter, TradeFilterItemDescriptions) {
		params.Set("get_descriptions", "1")
	}

	if testBit(filter, TradeFilterHistoricalOnly) {
		params.Set("historical_only", "1")
		params.Set("time_historical_cutoff", strconv.FormatInt(timeCutOff.Unix(), 10))
	}

	resp, err := session.client.Get(apiGetTradeOffers + params.Encode())
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

	return response.Inner, nil
}

func (session *Session) GetMyTradeToken() (string, error) {
	resp, err := session.client.Get("https://steamcommunity.com/my/tradeoffers/privacy")
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http error: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	m := offerInfoExp.FindStringSubmatch(string(body))
	if m == nil || len(m) != 2 {
		return "", ErrCannotFindOfferInfo
	}

	return string(m[1]), nil
}

type EscrowSteamGuardInfo struct {
	MyDays   int64
	ThemDays int64
	ErrorMsg string
}

func (session *Session) GetEscrowGuardInfo(sid SteamID, token string) (*EscrowSteamGuardInfo, error) {
	resp, err := session.client.Get("https://steamcommunity.com/tradeoffer/new/?" + url.Values{
		"partner": {strconv.FormatUint(uint64(sid.GetAccountID()), 10)},
		"token":   {token},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var my int64
	var them int64
	var errMsg string

	m := myEscrowExp.FindStringSubmatch(string(body))
	if m != nil && len(m) == 2 {
		my, _ = strconv.ParseInt(m[1], 10, 32)
	}

	m = themEscrowExp.FindStringSubmatch(string(body))
	if m != nil && len(m) == 2 {
		them, _ = strconv.ParseInt(m[1], 10, 32)
	}

	m = errorMsgExp.FindStringSubmatch(string(body))
	if m != nil && len(m) == 2 {
		errMsg = string(m[1])
	}

	return &EscrowSteamGuardInfo{
		MyDays:   my,
		ThemDays: them,
		ErrorMsg: errMsg,
	}, nil
}

func (session *Session) SendTradeOffer(offer *TradeOffer, sid SteamID, token string) error {
	content := map[string]interface{}{
		"newversion": true,
		"version":    3,
		"me": map[string]interface{}{
			"assets":   offer.SendItems,
			"currency": make([]struct{}, 0),
			"ready":    false,
		},
		"them": map[string]interface{}{
			"assets":   offer.RecvItems,
			"currency": make([]struct{}, 0),
			"ready":    false,
		},
	}

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://steamcommunity.com/tradeoffer/new/send",
		strings.NewReader(url.Values{
			"sessionid":                 {session.sessionID},
			"serverid":                  {"1"},
			"partner":                   {sid.ToString()},
			"tradeoffermessage":         {offer.Message},
			"json_tradeoffer":           {string(contentJSON)},
			"trade_offer_create_params": {"{\"trade_offer_access_token\":\"" + token + "\"}"},
		}.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Add("Referer", "https://steamcommunity.com/tradeoffer/new/?"+url.Values{
		"partner": {strconv.FormatUint(uint64(sid.GetAccountID()), 10)},
		"token":   {token},
	}.Encode())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := session.client.Do(req)
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

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if len(response.ErrorMessage) != 0 {
		return errors.New(response.ErrorMessage)
	}

	if response.ID == 0 {
		return errors.New("no OfferID included")
	}

	offer.ID = response.ID
	offer.Created = time.Now().Unix()
	offer.Updated = time.Now().Unix()
	offer.Expires = offer.Created + 14*24*60*60
	offer.RealTime = false
	offer.IsOurOffer = true

	// Just test mobile confirmation, email is deprecated
	if response.MobileConfirmationRequired {
		offer.ConfirmationMethod = TradeConfirmationMobileApp
		offer.State = TradeStateCreatedNeedsConfirmation
	} else {
		// set state to active
		offer.State = TradeStateActive
	}

	return nil
}

func (session *Session) GetTradeReceivedItems(receiptID uint64) ([]*InventoryItem, error) {
	resp, err := session.client.Get(fmt.Sprintf("https://steamcommunity.com/trade/%d/receipt", receiptID))
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	m := receiptExp.FindAllSubmatch(body, -1)
	if m == nil {
		return nil, ErrReceiptMatch
	}

	items := make([]*InventoryItem, len(m))
	for k := range m {
		item := &InventoryItem{}
		if err = json.Unmarshal(m[k][1], item); err != nil {
			return nil, err
		}

		items[k] = item
	}

	return items, nil
}

func (session *Session) DeclineTradeOffer(id uint64) error {
	resp, err := session.client.PostForm(apiDeclineTradeOffer, url.Values{
		"key":          {session.apiKey},
		"tradeofferid": {strconv.FormatUint(id, 10)},
	})
	if resp != nil {
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	result := resp.Header.Get("x-eresult")
	if result != "1" {
		return fmt.Errorf("cannot decline trade: %s", result)
	}

	return nil
}

func (session *Session) CancelTradeOffer(id uint64) error {
	resp, err := session.client.PostForm(apiCancelTradeOffer, url.Values{
		"key":          {session.apiKey},
		"tradeofferid": {strconv.FormatUint(id, 10)},
	})
	if resp != nil {
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	result := resp.Header.Get("x-eresult")
	if result != "1" {
		return fmt.Errorf("cannot cancel trade: %s", result)
	}

	return nil
}

func (session *Session) AcceptTradeOffer(id uint64) error {
	tid := strconv.FormatUint(id, 10)
	postURL := "https://steamcommunity.com/tradeoffer/" + tid

	req, err := http.NewRequest(
		http.MethodPost,
		postURL+"/accept",
		strings.NewReader(url.Values{
			"sessionid":    {session.sessionID},
			"serverid":     {"1"},
			"tradeofferid": {tid},
		}.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Add("Referer", postURL)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := session.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	type Response struct {
		ErrorMessage string `json:"strError"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if len(response.ErrorMessage) != 0 {
		return errors.New(response.ErrorMessage)
	}

	return nil
}

func (offer *TradeOffer) Send(session *Session, sid SteamID, token string) error {
	return session.SendTradeOffer(offer, sid, token)
}

func (offer *TradeOffer) Accept(session *Session) error {
	return session.AcceptTradeOffer(offer.ID)
}

func (offer *TradeOffer) Cancel(session *Session) error {
	if offer.IsOurOffer {
		return session.CancelTradeOffer(offer.ID)
	}

	return session.DeclineTradeOffer(offer.ID)
}

package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	CurrencyUSD = "1"
	CurrencyGBP = "2"
	CurrencyEUR = "3"
	CurrencyCHF = "4"
	CurrencyRUB = "5"
	CurrencyPLN = "6"
	CurrencyBRL = "7"
	CurrencyJPY = "8"
	CurrencyNOK = "9"
	CurrencyIDR = "10"
	CurrencyMYR = "11"
	CurrencyPHP = "12"
	CurrencySGD = "13"
	CurrencyTHB = "14"
	CurrencyVND = "15"
	CurrencyKRW = "16"
	CurrencyTRY = "17"
	CurrencyUAH = "18"
	CurrencyMXN = "19"
	CurrencyCAD = "20"
	CurrencyAUD = "21"
	CurrencyNZD = "22"
	CurrencyCNY = "23"
	CurrencyINR = "24"
	CurrencyCLP = "25"
	CurrencyPEN = "26"
	CurrencyCOP = "27"
	CurrencyZAR = "28"
	CurrencyHKD = "29"
	CurrencyTWD = "30"
	CurrencySAR = "31"
	CurrencyAED = "32"
)

type MarketItemPriceOverview struct {
	Success     bool   `json:"success"`
	LowestPrice string `json:"lowest_price"`
	MedianPrice string `json:"median_price"`
	Volume      string `json:"volume"`
}

type MarketItemPrice struct {
	Date  string
	Price float64
	Count string
}

type MarketItemResponse struct {
	Success     bool        `json:"success"`
	PricePrefix string      `json:"price_prefix"`
	PriceSuffix string      `json:"price_suffix"`
	Prices      interface{} `json:"prices"`
}

type MarketSellResponse struct {
	Success                    bool   `json:"success"`
	RequiresConfirmation       uint32 `json:"requires_confirmation"`
	MobileConfirmationRequired bool   `json:"needs_mobile_confirmation"`
	EmailConfirmationRequired  bool   `json:"needs_email_confirmation"`
	EmailDomain                string `json:"email_domain"`
}

var (
	ErrCannotLoadPrices     = errors.New("unable to load prices at this time")
	ErrInvalidPriceResponse = errors.New("invalid market pricehistory response")
)

func (session *Session) GetMarketItemPriceHistory(appID uint64, marketHashName string) ([]*MarketItemPrice, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/pricehistory/?" + url.Values{
		"appid":            {strconv.FormatUint(appID, 10)},
		"market_hash_name": {marketHashName},
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

	response := MarketItemResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if !response.Success {
		return nil, ErrCannotLoadPrices
	}

	var prices []interface{}
	var ok bool
	if prices, ok = response.Prices.([]interface{}); !ok {
		return nil, ErrCannotLoadPrices
	}

	items := []*MarketItemPrice{}
	for _, v := range prices {
		if v, ok := v.([]interface{}); ok {
			item := &MarketItemPrice{}
			for _, val := range v {
				switch val := val.(type) {
				case string:
					if len(item.Date) != 0 {
						item.Count = val
					} else {
						item.Date = val
					}
				case float64:
					item.Price = val
				}
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (session *Session) GetMarketItemPriceOverview(appID uint64, country, currencyID, marketHashName string) (*MarketItemPriceOverview, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/priceoverview/?" + url.Values{
		"appid":            {strconv.FormatUint(appID, 10)},
		"country":          {country},
		"currencyID":       {currencyID},
		"market_hash_name": {marketHashName},
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

	overview := &MarketItemPriceOverview{}
	if err = json.NewDecoder(resp.Body).Decode(overview); err != nil {
		return nil, err
	}

	return overview, nil
}

func (session *Session) SellItem(item *InventoryItem, amount, price uint64) (*MarketSellResponse, error) {
	resp, err := session.client.PostForm("https://steamcommunity.com/market/sellitem/", url.Values{
		"amount":    {strconv.FormatUint(amount, 10)},
		"appid":     {strconv.FormatUint(item.AppID, 10)},
		"assetid":   {strconv.FormatUint(item.AssetID, 10)},
		"contextid": {strconv.FormatUint(item.ContextID, 10)},
		"price":     {strconv.FormatUint(price, 10)},
		"sessionid": {session.sessionID},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	response := &MarketSellResponse{}
	if err = json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return response, nil
}

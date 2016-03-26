package steam

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
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

var (
	ErrCannotLoadPrices     = errors.New("unable to load prices at this time")
	ErrInvalidPriceResponse = errors.New("invalid market pricehistory response")
)

func (session *Session) GetMarketItemPriceHistory(appID uint16, marketHashName string) ([]*MarketItemPrice, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/pricehistory/?" + url.Values{
		"appid":            {strconv.FormatUint(uint64(appID), 10)},
		"market_hash_name": {marketHashName},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
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

func (session *Session) GetMarketItemPriceOverview(appID uint16, marketHashName string) (*MarketItemPriceOverview, error) {
	resp, err := session.client.Get("https://steamcommunity.com/market/priceoverview/?" + url.Values{
		"appid":            {strconv.FormatUint(uint64(appID), 10)},
		"market_hash_name": {marketHashName},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	overview := &MarketItemPriceOverview{}
	if err = json.NewDecoder(resp.Body).Decode(overview); err != nil {
		return nil, err
	}

	return overview, nil
}

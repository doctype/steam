/**
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
	"net/http"
	"net/url"
)

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

var ErrCannotLoadPrices = errors.New("unable to load prices at this time")

func (community *Community) GetMarketItemPriceHistory(appid uint16, marketHashName string) ([]*MarketItemPrice, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://steamcommunity.com/market/pricehistory/?appid=%d&market_hash_name=%s",
			appid, url.QueryEscape(marketHashName)),
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := community.client.Do(req)
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

	switch response.Prices.(type) {
	case []interface{}:
		items := []*MarketItemPrice{}
		for _, v := range response.Prices.([]interface{}) {
			switch v.(type) {
			case []interface{}:
				d := v.([]interface{})
				items = append(items, &MarketItemPrice{
					Date:  d[0].(string),
					Price: d[1].(float64),
					Count: d[2].(string),
				})
			default:
				// ignore
			}
		}

		return items, nil
	case bool:
		return nil, ErrCannotLoadPrices
	}

	return nil, fmt.Errorf("GetMarketItemPriceHistory(): please implement type handler for %v", response.Prices)
}

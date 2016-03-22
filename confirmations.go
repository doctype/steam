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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Confirmation struct {
	ID        uint64
	Key       uint64
	Title     string
	Receiving string
	Since     string
}

var (
	OfferIDPart = "tradeofferid_"

	ErrCannotFindConfirmations   = errors.New("unable to find confirmations")
	ErrCannotFindDescriptions    = errors.New("unable to find confirmation descriptions")
	ErrConfiramtionsDescMismatch = errors.New("cannot match confirmations with their respective descriptions")
	ErrConfirmationOfferIDFail   = errors.New("unable to get confirmation offer id")
	ErrCannotFindTradeOffer      = errors.New("unable to find tradeoffer div to get offer id for confirmation")
	ErrCannotFindOfferIDAttr     = errors.New("unable to find offer ID attribute")
)

func (community *Community) execConfirmationRequest(request, key, tag string, current int64) (*http.Response, error) {
	params := url.Values{
		"p":   {community.deviceID},
		"a":   {community.steamID.ToString()},
		"k":   {key},
		"t":   {strconv.FormatInt(current, 10)},
		"m":   {"android"},
		"tag": {tag},
	}

	req, err := http.NewRequest(http.MethodGet, "https://steamcommunity.com/mobileconf/"+request+"?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	return community.client.Do(req)
}

func (community *Community) GetConfirmations(key string) ([]*Confirmation, error) {
	resp, err := community.execConfirmationRequest("conf", key, "conf", time.Now().Unix())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(io.Reader(resp.Body))
	if err != nil {
		return nil, err
	}

	entries := doc.Find(".mobileconf_list_entry")
	if entries == nil {
		return nil, ErrCannotFindConfirmations
	}

	descriptions := doc.Find(".mobileconf_list_entry_description")
	if descriptions == nil {
		return nil, ErrCannotFindDescriptions
	}

	if len(entries.Nodes) != len(descriptions.Nodes) {
		return nil, ErrConfiramtionsDescMismatch
	}

	confirmations := []*Confirmation{}
	for k, sel := range entries.Nodes {
		confirmation := &Confirmation{}
		for _, attr := range sel.Attr {
			if attr.Key == "data-confid" {
				confirmation.ID, _ = strconv.ParseUint(attr.Val, 10, 32)
			} else if attr.Key == "data-key" {
				confirmation.Key, _ = strconv.ParseUint(attr.Val, 10, 64)
			}
		}

		descSel := descriptions.Nodes[k]
		depth := 0
		for child := descSel.FirstChild; child != nil; child = child.NextSibling {
			for n := child.FirstChild; n != nil; n = n.NextSibling {
				switch depth {
				case 0:
					confirmation.Title = n.Data
				case 1:
					confirmation.Receiving = n.Data
				case 2:
					confirmation.Since = n.Data
				}
				depth++
			}
		}

		confirmations = append(confirmations, confirmation)
	}

	return confirmations, nil
}

func (community *Community) GetConfirmationOfferID(key string, cid uint64) (uint64, error) {
	resp, err := community.execConfirmationRequest(fmt.Sprintf("details/%d", cid), key, "details", time.Now().Unix())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return 0, err
	}

	type Response struct {
		Success bool   `json:"success"`
		Html    string `json:"html"`
	}

	var r Response
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return 0, err
	}

	if !r.Success {
		return 0, ErrConfirmationOfferIDFail
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(r.Html))
	if err != nil {
		return 0, err
	}

	offer := doc.Find(".tradeoffer")
	if offer == nil {
		return 0, ErrCannotFindTradeOffer
	}

	for _, sel := range offer.Nodes {
		for _, attr := range sel.Attr {
			if attr.Key == "id" {
				val := attr.Val
				if len(val) <= len(OfferIDPart) || val[:len(OfferIDPart)] != OfferIDPart {
					// ?
					continue
				}

				id := val[len(OfferIDPart):]
				raw, err := strconv.ParseUint(id, 10, 64)
				if err != nil {
					return 0, err
				}

				return raw, nil
			}
		}
	}

	return 0, ErrCannotFindOfferIDAttr
}

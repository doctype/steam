package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// Due to the JSON being string, etc... we cannot re-use EconItem
// Also, "assetid" is included as "id" not as assetid.
type InventoryItem struct {
	AssetID        uint64 `json:"id,string,omitempty"`
	InstanceID     uint64 `json:"instanceid,string,omitempty"`
	ClassID        uint64 `json:"classid,string,omitempty"`
	AppID          uint32 `json:"appid"`     // This!
	ContextID      uint16 `json:"contextid"` // Ditto
	Name           string `json:"name"`
	MarketHashName string `json:"market_hash_name"`
}

var ErrCannotLoadInventory = errors.New("unable to load inventory at this time")

func (community *Community) parseInventory(sid *SteamID, appid, contextid, start uint32, tradableOnly bool, items *[]*InventoryItem) (uint32, error) {
	url := "https://steamcommunity.com/profiles/%d/inventory/json/%d/%d/?start=%d"
	if tradableOnly {
		url += "&trading=1"
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(url, *sid, appid, contextid, start), nil)
	if err != nil {
		return 0, err
	}

	resp, err := community.client.Do(req)
	if err != nil {
		return 0, err
	}

	type DescItem struct {
		Name           string `json:"name"`
		MarketHashName string `json:"market_hash_name"`
	}

	type Response struct {
		Success      bool                      `json:"success"`
		MoreStart    interface{}               `json:"more_start"` // This can be a bool or a number...
		Inventory    map[string]*InventoryItem `json:"rgInventory"`
		Descriptions map[string]*DescItem      `json:"rgDescriptions"`
		/* Missing: rgCurrency  */
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	if !response.Success {
		return 0, ErrCannotLoadInventory
	}

	// Morph response.Inventory into an array of items.
	// This is due to Steam returning the items in the following format:
	//	rgInventory: {
	//		"54xxx": {
	//			"id": "54xxx"
	//			...
	//		}
	//	}
	for _, value := range response.Inventory {
		desc, ok := response.Descriptions[strconv.FormatUint(value.ClassID, 10)+"_"+strconv.FormatUint(value.InstanceID, 10)]
		if ok {
			value.Name = desc.Name
			value.MarketHashName = desc.MarketHashName
		}

		*items = append(*items, value)
	}

	switch response.MoreStart.(type) {
	case int, uint:
		return uint32(response.MoreStart.(int)), nil
	case bool:
		break
	default:
		return 0, fmt.Errorf("parseInventory(): Please implement case for type %v", response.MoreStart)
	}

	return 0, nil
}

func (community *Community) GetInventory(sid *SteamID, appid, contextid uint32, tradableOnly bool) ([]*InventoryItem, error) {
	items := []*InventoryItem{}
	more := uint32(0)

	for {
		next, err := community.parseInventory(sid, appid, contextid, more, tradableOnly, &items)
		if err != nil {
			return nil, err
		}

		if next == 0 {
			break
		}

		more = next
	}

	return items, nil
}

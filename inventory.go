package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
)

// Due to the JSON being string, etc... we cannot re-use EconItem
// Also, "assetid" is included as "id" not as assetid.
type InventoryItem struct {
	AssetID    uint64        `json:"id,string,omitempty"`
	InstanceID uint64        `json:"instanceid,string,omitempty"`
	ClassID    uint64        `json:"classid,string,omitempty"`
	AppID      uint64        `json:"appid"`     // This!  (May be null; see desc if so)
	ContextID  uint64        `json:"contextid"` // Ditto  (May be null; see desc if so)
	Desc       *EconItemDesc `json:"-"`
}

type InventoryContext struct {
	ID         uint64 `json:"id,string"` /* Apparently context id needs at least 64 bits...  */
	AssetCount uint32 `json:"asset_count"`
	Name       string `json:"name"`
}

type InventoryAppStats struct {
	AppID            uint64                       `json:"appid"`
	Name             string                       `json:"name"`
	AssetCount       uint32                       `json:"asset_count"`
	Icon             string                       `json:"icon"`
	Link             string                       `json:"link"`
	InventoryLogo    string                       `json:"inventory_logo"`
	TradePermissions string                       `json:"trade_permissions"`
	Contexts         map[string]*InventoryContext `json:"rgContexts"`
}

var inventoryContextRegexp = regexp.MustCompile("var g_rgAppContextData = (.*?);")

func (session *Session) parseInventory(sid SteamID, appID, contextID uint64, start uint32, tradableOnly bool, items *[]*InventoryItem) (uint32, error) {
	params := url.Values{
		"start": {strconv.FormatUint(uint64(start), 10)},
	}
	if tradableOnly {
		params.Set("trading", "1")
	}

	resp, err := session.client.Get(fmt.Sprintf("https://steamcommunity.com/profiles/%d/inventory/json/%d/%d/?", sid, appID, contextID) + params.Encode())
	if err != nil {
		return 0, err
	}

	type Response struct {
		Success      bool            `json:"success"`
		ErrorMsg     string          `json:"Error"`
		MoreStart    interface{}     `json:"more_start"` // This can be a bool or a number...
		Inventory    json.RawMessage `json:"rgInventory"`
		Descriptions json.RawMessage `json:"rgDescriptions"`
		/* Missing: rgCurrency  */
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, err
	}

	if !response.Success {
		return 0, errors.New(response.ErrorMsg)
	}

	var inventory map[string]*InventoryItem
	if err = json.Unmarshal(response.Inventory, &inventory); err != nil {
		// empty inventory...
		// NB: This only occurs on first run...
		return 0, nil
	}

	var descriptions map[string]*EconItemDesc
	if err = json.Unmarshal(response.Descriptions, &descriptions); err != nil {
		if inventory != nil && len(inventory) != 0 {
			return 0, err
		}

		return 0, nil
	}

	// Morph inventory into an array of items.
	// This is due to Steam returning the items in the following format:
	//	rgInventory: {
	//		"54xxx": {
	//			"id": "54xxx"
	//			...
	//		}
	//	}
	// We also glue the descriptions.
	for _, value := range inventory {
		if desc, ok := descriptions[strconv.FormatUint(value.ClassID, 10)+"_"+strconv.FormatUint(value.InstanceID, 10)]; ok {
			value.Desc = desc
		}

		*items = append(*items, value)
	}

	io.Copy(ioutil.Discard, resp.Body)
	switch v := response.MoreStart.(type) {
	case int:
		return uint32(v), nil
	case uint:
		return uint32(v), nil
	case bool:
		break
	default:
		return 0, fmt.Errorf("parseInventory: missing implementation for type %v", v)
	}

	return 0, nil
}

func (session *Session) GetInventory(sid SteamID, appID, contextID uint64, tradableOnly bool) ([]*InventoryItem, error) {
	items := []*InventoryItem{}
	more := uint32(0)

	for {
		next, err := session.parseInventory(sid, appID, contextID, more, tradableOnly, &items)
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

func (session *Session) GetInventoryAppStats(sid SteamID) (map[string]InventoryAppStats, error) {
	resp, err := session.client.Get("https://steamcommunity.com/profiles/" + sid.ToString() + "/inventory")
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

	m := inventoryContextRegexp.FindSubmatch(body)
	if m == nil || len(m) != 2 {
		return nil, err
	}

	inven := map[string]InventoryAppStats{}
	if err = json.Unmarshal(m[1], &inven); err != nil {
		return nil, err
	}

	return inven, nil
}

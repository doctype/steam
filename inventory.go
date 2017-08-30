package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
)

const (
	InventoryEndpoint = "http://steamcommunity.com/inventory/%d/%d/%d?"
)

type ItemTag struct {
	Category              string `json:"category"`
	InternalName          string `json:"internal_name"`
	LocalizedCategoryName string `json:"localized_category_name"`
	LocalizedTagName      string `json:"localized_tag_name"`
}

// Due to the JSON being string, etc... we cannot re-use EconItem
// Also, "assetid" is included as "id" not as assetid.
type InventoryItem struct {
	AppID      uint32        `json:"appid"`
	ContextID  uint64        `json:"contextid"`
	AssetID    uint64        `json:"id,string,omitempty"`
	ClassID    uint64        `json:"classid,string,omitempty"`
	InstanceID uint64        `json:"instanceid,string,omitempty"`
	Amount     uint64        `json:"amount,string"`
	Desc       *EconItemDesc `json:"-"` /* May be nil  */
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

func (session *Session) fetchInventory(
	sid SteamID,
	appID, contextID, startAssetID uint64,
	filters []Filter,
	items *[]InventoryItem,
) (hasMore bool, lastAssetID uint64, err error) {
	params := url.Values{
		"l": {session.language},
	}

	if startAssetID != 0 {
		params.Set("start_assetid", strconv.FormatUint(startAssetID, 10))
		params.Set("count", "75")
	} else {
		params.Set("count", "250")
	}

	resp, err := session.client.Get(fmt.Sprintf(InventoryEndpoint, sid, appID, contextID) + params.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return false, 0, err
	}

	type Asset struct {
		AppID      uint32 `json:"appid"`
		ContextID  uint64 `json:"contextid,string"`
		AssetID    uint64 `json:"assetid,string"`
		ClassID    uint64 `json:"classid,string"`
		InstanceID uint64 `json:"instanceid,string"`
		Amount     uint64 `json:"amount,string"`
	}

	type Response struct {
		Assets              []Asset         `json:"assets"`
		Descriptions        []*EconItemDesc `json:"descriptions"`
		Success             int             `json:"success"`
		HasMore             int             `json:"more_items"`
		LastAssetID         string          `json:"last_assetid"`
		TotalInventoryCount int             `json:"total_inventory_count"`
		ErrorMsg            string          `json:"error"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, 0, err
	}

	if response.Success == 0 {
		if len(response.ErrorMsg) != 0 {
			return false, 0, errors.New(response.ErrorMsg)
		}

		return false, 0, nil // empty inventory
	}

	// Fill in descriptions map, where key
	// is "<CLASS_ID>_<INSTANCE_ID>" pattern, and
	// value is position on asset description in
	// response.Descriptions array
	//
	// We need it for fast asset's description
	// searching in future
	descriptions := make(map[string]int)
	for i, desc := range response.Descriptions {
		key := fmt.Sprintf("%d_%d", desc.ClassID, desc.InstanceID)
		descriptions[key] = i
	}

	for _, asset := range response.Assets {
		var desc *EconItemDesc

		key := fmt.Sprintf("%d_%d", asset.ClassID, asset.InstanceID)
		if d, ok := descriptions[key]; ok {
			desc = response.Descriptions[d]
		}

		item := InventoryItem{
			AppID:      asset.AppID,
			ContextID:  asset.ContextID,
			AssetID:    asset.AssetID,
			ClassID:    asset.ClassID,
			InstanceID: asset.InstanceID,
			Amount:     asset.Amount,
			Desc:       desc,
		}

		add := true
		for _, filter := range filters {
			add = filter(&item)
			if !add {
				break
			}
		}

		if add {
			*items = append(*items, item)
		}
	}

	hasMore = response.HasMore != 0
	if !hasMore {
		return hasMore, 0, nil
	}

	lastAssetID, err = strconv.ParseUint(response.LastAssetID, 10, 64)
	if err != nil {
		return hasMore, 0, err
	}

	return hasMore, lastAssetID, nil
}

func (session *Session) GetInventory(sid SteamID, appID, contextID uint64, tradableOnly bool) ([]InventoryItem, error) {
	filters := []Filter{}

	if tradableOnly {
		filters = append(filters, IsTradable(tradableOnly))
	}

	return session.GetFilterableInventory(sid, appID, contextID, filters)
}

func (session *Session) GetFilterableInventory(sid SteamID, appID, contextID uint64, filters []Filter) ([]InventoryItem, error) {
	items := []InventoryItem{}
	startAssetID := uint64(0)

	for {
		hasMore, lastAssetID, err := session.fetchInventory(sid, appID, contextID, startAssetID, filters, &items)
		if err != nil {
			return nil, err
		}

		if !hasMore {
			break
		}

		startAssetID = lastAssetID
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

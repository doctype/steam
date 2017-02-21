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

type Lang string

const (
	LangNone = "none"
	LangEng  = "english"
	LangRus  = "russian"
)

const (
	InventoryEndpoint = "http://steamcommunity.com/inventory/%d/%d/%d"
)

type ItemPict struct {
	Standard string `json:"standard"`
	Large    string `json:"large"`
}

type ItemName struct {
	Text       string `json:"text"`
	Color      string `json:"color"`
	MarketText string `json:"market_text"`
	MarketHash string `json:"market_hash"`
}

type ItemMarket struct {
	Marketable   bool `json:"marketable"`
	Restrictions int  `json:"restrictions"`
}

// Due to the JSON being string, etc... we cannot re-use EconItem
// Also, "assetid" is included as "id" not as assetid.
type InventoryItem struct {
	AppID       uint64     `json:"appid"`
	ContextID   uint64     `json:"contextid"`
	AssetID     uint64     `json:"assetid"`
	ClassID     uint64     `json:"classid"`
	InstanceID  uint64     `json:"instanceid"`
	Amount      uint64     `json:"amount"`
	Tradable    bool       `json:"tradable"`
	Currency    int        `json:"currency"`
	Pictures    ItemPict   `json:"pictures"`
	Description string     `json:"desc"`
	Name        ItemName   `json:"name"`
	Commodity   bool       `json:"commodity"`
	Type        string     `json:"type"`
	Market      ItemMarket `json:"market"`
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
	appID, contextID uint64,
	lang Lang,
	startAssetID uint64,
	items *[]InventoryItem,
	filters *[]Filter,
) (hasMore bool, lastAssetID uint64, err error) {
	params := url.Values{}

	if lang != LangNone {
		params.Set("l", string(lang))
	}

	if startAssetID > 0 {
		params.Set("start_assetid", strconv.FormatUint(startAssetID, 10))
	}

	requestURL := fmt.Sprintf(InventoryEndpoint, sid, appID, contextID)

	resp, err := session.client.Get(requestURL + "?" + params.Encode())
	if err != nil {
		return false, 0, err
	}

	type Asset struct {
		AppID      uint64 `json:"appid,string"`
		ContextID  uint64 `json:"contextid,string"`
		AssetID    uint64 `json:"assetid,string"`
		ClassID    uint64 `json:"classid,string"`
		InstanceID uint64 `json:"instanceid,string"`
		Amount     uint64 `json:"amount,string"`
	}

	type DescriptionsPart struct {
		Type  string `json:"type"`
		Value string `json:"value"`
		Color string `json:"color"`
	}

	type Description struct {
		AppID                     uint64             `json:"appid"`
		ClassID                   uint64             `json:"classid,string"`
		InstanceID                uint64             `json:"instanceid,string"`
		Currency                  int                `json:"currency"`
		BackgroundColor           string             `json:"background_color"`
		IconURL                   string             `json:"icon_url"`
		IconURLLarge              string             `json:"icon_url_large"`
		Descriptions              []DescriptionsPart `json:"descriptions"`
		Tradable                  int                `json:"tradable"`
		Name                      string             `json:"name"`
		NameColor                 string             `json:"name_color"`
		Type                      string             `json:"type"`
		MarketName                string             `json:"market_name"`
		MarketHashName            string             `json:"market_hash_name"`
		Commodity                 int                `json:"commodity"`
		MarketTradableRestriction int                `json:"market_tradable_restriction"`
		Marketable                int                `json:"marketable"`
	}

	type Response struct {
		Assets              []Asset       `json:"assets"`
		Descriptions        []Description `json:"descriptions"`
		Success             int           `json:"success"`
		HasMore             int           `json:"more_items"`
		LastAssetID         string        `json:"last_assetid"`
		TotalInventoryCount int           `json:"total_inventory_count"`
		ErrorMsg            string        `json:"error"`
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

	descriptions := make(map[string]int)

	// Fill in descriptions map, where key
	// is "<CLASS_ID>_<INSTANCE_ID>" pattern, and
	// value is position on asset description in
	// response.Descriptions array
	//
	// We need it for fast asset's description
	// searching in future
	for i, desc := range response.Descriptions {
		key := fmt.Sprintf("%d_%d", desc.ClassID, desc.InstanceID)

		descriptions[key] = i
	}

	for _, asset := range response.Assets {
		key := fmt.Sprintf("%d_%d", asset.ClassID, asset.InstanceID)
		descPos := descriptions[key]

		picts := ItemPict{
			Standard: response.Descriptions[descPos].IconURL,
		}

		if len(response.Descriptions[descPos].IconURLLarge) != 0 {
			picts.Large = response.Descriptions[descPos].IconURLLarge
		}

		item := InventoryItem{
			AppID:       asset.AppID,
			ContextID:   asset.ContextID,
			AssetID:     asset.AssetID,
			ClassID:     asset.ClassID,
			InstanceID:  asset.InstanceID,
			Amount:      asset.Amount,
			Tradable:    response.Descriptions[descPos].Tradable == 1,
			Currency:    response.Descriptions[descPos].Currency,
			Pictures:    picts,
			Description: "",
			Name: ItemName{
				Text:       response.Descriptions[descPos].Name,
				Color:      response.Descriptions[descPos].NameColor,
				MarketText: response.Descriptions[descPos].MarketName,
				MarketHash: response.Descriptions[descPos].MarketHashName,
			},
			Commodity: response.Descriptions[descPos].Commodity == 1,
			Type:      response.Descriptions[descPos].Type,
			Market: ItemMarket{
				Marketable:   response.Descriptions[descPos].Marketable == 1,
				Restrictions: response.Descriptions[descPos].MarketTradableRestriction,
			},
		}

		st := true
		for _, filter := range *filters {
			if !filter(item) {
				st = false

				break
			}
		}

		if !st {
			continue
		}

		*items = append(*items, item)
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

func (session *Session) GetInventory(
	sid SteamID,
	appID, contextID uint64,
	tradableOnly bool,
) ([]InventoryItem, error) {
	filters := []Filter{}

	if tradableOnly {
		filters = append(filters, IsTradable(tradableOnly))
	}

	return session.GetInventoryInternal(sid, 730, 2, LangNone, filters)
}

func (session *Session) GetInventoryInternal(
	sid SteamID,
	appID,
	contextID uint64,
	lang Lang,
	filters []Filter,
) ([]InventoryItem, error) {
	items := []InventoryItem{}
	startAssetID := uint64(0)

	for {
		hasMore, lastAssetID, err := session.fetchInventory(sid, appID, contextID, lang, startAssetID, &items, &filters)
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

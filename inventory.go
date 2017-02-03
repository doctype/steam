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
	LangEng = "english"
	LangRus = "russian"
)

const (
	PicturePathPattern = "http://steamcommunity-a.akamaihd.net/economy/image/%s"
)

type ItemPict struct {
	Standart string `json:"standart"`
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

func (session *Session) fetchInventory(sid SteamID, appID, contextID uint64, lang Lang, startAssetID uint64, items *[]InventoryItem) (hasMore bool, lastAssetID uint64, err error) {
	params := url.Values{
		"l":             {string(lang)},
		"start_assetid": {strconv.FormatUint(startAssetID, 10)},
	}

	requestURL := fmt.Sprintf("http://steamcommunity.com/inventory/%d/%d/%d", sid, appID, contextID)

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

	// New inventory endpoint has same order for "Assets" and "Descriptions"
	// So we don't need create Descriptions map
	// Now we can just incriment Descriptions array iterator with Assets iterator
	// but when current Asset ClassID and InstanceID are equal with previos
	// we mustn't incriment Descriptions iterator
	//
	// It looks like:
	// Assets  descPos
	//   A        0
	//   A        0
	//   B        1
	//   C        2
	//   C        2
	//   C        2
	//   D        3
	descPos := 0
	lastKey := fmt.Sprintf("%d_%d", response.Assets[0].ClassID, response.Assets[0].InstanceID)
	for _, asset := range response.Assets {
		tmpKey := fmt.Sprintf("%d_%d", asset.ClassID, asset.InstanceID)
		if tmpKey != lastKey {
			lastKey = tmpKey

			descPos++
		}

		picts := ItemPict{
			Standart: fmt.Sprintf(PicturePathPattern, response.Descriptions[descPos].IconURL),
		}

		if len(response.Descriptions[descPos].IconURLLarge) != 0 {
			picts.Large = fmt.Sprintf(PicturePathPattern, response.Descriptions[descPos].IconURLLarge)
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

		*items = append(*items, item)
	}

	hasMore = response.HasMore == 1

	if !hasMore {
		return hasMore, 0, nil
	}

	lastAssetID, err = strconv.ParseUint(response.LastAssetID, 10, 64)
	if err != nil {
		return hasMore, 0, err
	}

	return hasMore, lastAssetID, nil
}

func (session *Session) GetInventory(sid SteamID, appID, contextID uint64, lang Lang) ([]InventoryItem, error) {
	items := []InventoryItem{}
	startAssetID := uint64(0)

	for {
		hasMore, lastAssetID, err := session.fetchInventory(sid, appID, contextID, lang, startAssetID, &items)
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

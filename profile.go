package steam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const (
	PrivacyStatePrivate     = 1
	PrivacyStateFriendsOnly = 2
	PrivacyStatePublic      = 3
)

const (
	CommentSettingSelf    = "commentselfonly"
	CommentSettingFriends = "commentfriendsonly"
	CommentSettingPublic  = "commentanyone"
)

const (
	apiGetPlayerSummaries = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?"
	apiGetOwnedGames      = "https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?"
	apiGetPlayerBans      = "https://api.steampowered.com/ISteamUser/GetPlayerBans/v1/?"
)

type PlayerSummary struct {
	SteamID           SteamID `json:"steamid,string"`
	VisibilityState   uint32  `json:"communityvisibilitystate"`
	ProfileState      uint32  `json:"profilestate"`
	PersonaName       string  `json:"personaname"`
	PersonaState      uint32  `json:"personastate"`
	PersonaStateFlags uint32  `json:"personastateflags"`
	RealName          string  `json:"realname"`
	LastLogoff        int64   `json:"lastlogoff"`
	ProfileURL        string  `json:"profileurl"`
	AvatarURL         string  `json:"avatar"`
	AvatarMediumURL   string  `json:"avatarmedium"`
	AvatarFullURL     string  `json:"avatarfull"`
	PrimaryClanID     uint64  `json:"primaryclanid,string"`
	TimeCreated       int64   `json:"timecreated"`
	LocCountryCode    string  `json:"loccountrycode"`
	LocStateCode      string  `json:"locstatecode"`
	LocCityID         uint32  `json:"loccityid"`
}

type Game struct {
	AppID           uint32 `json:"appid"`
	PlaytimeForever int64  `json:"playtime_forever"`
	Playtime2Weeks  int64  `json:"playtime_2weeks"`
}

type OwnedGamesResponse struct {
	Count uint32  `json:"game_count"`
	Games []*Game `json:"games"`
}

type PlayerBan struct {
	SteamID          string `json:"SteamId"`
	CommunityBanned  bool   `json:"CommunityBanned"`
	VACBanned        bool   `json:"VACBanned"`
	NumberOfVACBans  int    `json:"NumberOfVACBans"`
	DaysSinceLastBan int    `json:"DaysSinceListBan"`
	NumberOfGameBans int    `json:"NumberOfGameBans"`
	EconomyBan       string `json:"EconomyBan"`
}

func (session *Session) GetProfileURL() (string, error) {
	tmpClient := http.Client{Jar: session.client.Jar}

	/* We do not follow redirect, we want to know where it'd redirect us.  */
	tmpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return errors.New("do not redirect")
	}

	/* Query normal, this will redirect us.  */
	resp, err := tmpClient.Get("https://steamcommunity.com/my")
	if resp == nil {
		return "", err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("http error: %d", resp.StatusCode)
	}

	/* We now have a few useful variables in header, for now, we will just grap "Location".  */
	return resp.Header.Get("Location"), nil
}

func (session *Session) SetupProfile(profileURL string) error {
	resp, err := session.client.Get(profileURL + "/edit?welcomed=1")
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return nil
}

func (session *Session) SetProfileInfo(profileURL string, values *map[string][]string) error {
	(*values)["sessionID"] = []string{session.sessionID}
	(*values)["type"] = []string{"profileSave"}

	resp, err := session.client.PostForm(profileURL+"/edit", *values)
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return nil
}

func (session *Session) SetProfilePrivacy(profileURL string, commentPrivacy string, privacy uint8) error {
	resp, err := session.client.PostForm(profileURL+"/edit/settings", url.Values{
		"sessionID":               {session.sessionID},
		"type":                    {"profileSettings"},
		"commentSetting":          {commentPrivacy},
		"privacySetting":          {strconv.FormatUint(uint64(privacy&0x3), 10)},
		"inventoryPrivacySetting": {strconv.FormatUint(uint64((privacy>>2)&0x3), 10)},
		"inventoryGiftPrivacy":    {strconv.FormatUint(uint64((privacy>>4)&0x3), 10)},
	})
	if resp != nil {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	return nil
}

func (session *Session) GetPlayerSummaries(steamids string) ([]*PlayerSummary, error) {
	resp, err := session.client.Get(apiGetPlayerSummaries + url.Values{
		"key":      {session.apiKey},
		"steamids": {steamids},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Players struct {
		Summaries []*PlayerSummary `json:"players"`
	}

	type Response struct {
		Inner Players `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner.Summaries, nil
}

func (session *Session) GetOwnedGames(sid SteamID, freeGames bool, appInfo bool) (*OwnedGamesResponse, error) {
	resp, err := session.client.Get(apiGetOwnedGames + url.Values{
		"key":                       {session.apiKey},
		"steamid":                   {sid.ToString()},
		"format":                    {"json"},
		"include_appinfo":           {strconv.FormatBool(appInfo)},
		"include_played_free_games": {strconv.FormatBool(freeGames)},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner *OwnedGamesResponse `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

func (session *Session) GetPlayerBans(steamids string) ([]*PlayerBan, error) {
	resp, err := session.client.Get(apiGetPlayerBans + url.Values{
		"key":      {session.apiKey},
		"steamids": {steamids},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner []*PlayerBan `json:"players"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

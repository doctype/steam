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
	PersonaStateOffline = iota
	PersonaStateOnline
	PersonaStateBusy
	PersonaStateAway
	PersonaStateSnooze
	PersonaStateLookingToTrade
	PersonaStateLookingToPlay
)

const (
	PersonaStateFlagRichPresence   = 1 << 0
	PersonaStateFlagInJoinableGame = 1 << 1
	PersonaStateFlagWeb            = 1 << 8
	PersonaStateFlagMobile         = 1 << 9
	PersonaStateFlagBigPicture     = 1 << 10
)

const (
	MessageTypeStatus  = "personastate"
	MessageTypeTyping  = "typing"
	MessageTypeSayText = "saytext"
)

const (
	ChatUIModeMobile = "mobile" // empty string works too
	ChatUIModeWeb    = "web"
)

const (
	apiUserPresenceLogin   = "https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logon/v1"
	apiUserPresenceLogoff  = "https://api.steampowered.com/ISteamWebUserPresenceOAuth/Logoff/v1"
	apiUserPresencePoll    = "https://api.steampowered.com/ISteamWebUserPresenceOAuth/Poll/v1"
	apiUserPresenceMessage = "https://api.steampowered.com/ISteamWebUserPresenceOAuth/Message/v1"
)

type ChatMessage struct {
	Type         string `json:"type"`
	Text         string `json:"text"`
	TimestampOff int64  `json:"timestamp"`
	UTCTimestamp int64  `json:"utc_timestamp"`
	Partner      uint32 `json:"accountid_from"`
	StatusFlags  uint32 `json:"status_flags"`
	PersonaState uint32 `json:"persona_state"`
	PersonaName  string `json:"persona_name"`
}

type ChatLogMessage struct {
	Partner   uint32 `json:"m_unAccountID"`
	Timestamp int64  `json:"m_tsTimestamp"`
	Message   string `json:"m_strMessage"`
}

type ChatResponse struct {
	Message      int            `json:"message"`       // Login / Internal
	UmqID        string         `json:"umqid"`         // Login / Internal
	TimestampOff int64          `json:"timestamp"`     // Login
	UTCTimestamp int64          `json:"utc_timestamp"` // Login
	Push         int            `json:"push"`          // Login
	ErrorMessage string         `json:"error"`         // All (returned as error if not "OK")
	MessageBase  uint32         `json:"messagebase"`   // ChatPoll
	LastMessages uint32         `json:"messagelast"`   // ChatPoll
	Messages     []*ChatMessage `json:"messages"`      // ChatPoll
	SecTimeout   uint32         `json:"sectimeout"`    // ChatPoll
}

type ChatFriendResponse struct {
	AccountID   uint32  `json:"m_unAccountID"`
	SteamID     SteamID `json:"m_ulSteamID,string"`
	Name        string  `json:"m_strName"`
	State       uint8   `json:"m_ePersonaState"`
	StateFlags  uint32  `json:"m_nPersonaStateFlags"`
	AvatarHash  string  `json:"m_strAvatarHash"`
	InGame      bool    `json:"m_bIngame"`
	InGameAppID uint64  `json:"m_nInGameAppID,string"`
	InGameName  string  `json:"m_strInGameName"`
	LastMessage int64   `json:"m_tsLastMessage"`
	LastView    int64   `json:"m_tsLastView"`
}

func (session *Session) ChatLogin(uiMode string) error {
	resp, err := session.client.PostForm(apiUserPresenceLogin, url.Values{
		"ui_mode":      {uiMode},
		"access_token": {session.oauth.Token},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error: %d", resp.StatusCode)
	}

	var response ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if response.ErrorMessage != "OK" {
		return errors.New(response.ErrorMessage)
	}

	session.umqID = response.UmqID
	session.chatMessage = response.Message
	return nil
}

func (session *Session) ChatLogoff() error {
	resp, err := session.client.PostForm(apiUserPresenceLogoff, url.Values{
		"access_token": {session.oauth.Token},
		"umqid":        {session.umqID},
	})
	if resp != nil {
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

func (session *Session) ChatSendMessage(sid SteamID, message, messageType string) error {
	resp, err := session.client.PostForm(apiUserPresenceMessage, url.Values{
		"access_token": {session.oauth.Token},
		"steamid_dst":  {sid.ToString()},
		"text":         {message},
		"type":         {messageType},
		"umqid":        {session.umqID},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var response ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if response.ErrorMessage != "OK" {
		return errors.New(response.ErrorMessage)
	}

	return nil
}

func (session *Session) ChatPoll(timeoutSeconds string) (*ChatResponse, error) {
	resp, err := session.client.PostForm(apiUserPresencePoll, url.Values{
		"umqid":          {session.umqID},
		"access_token":   {session.oauth.Token},
		"message":        {strconv.FormatUint(uint64(session.chatMessage), 10)},
		"pollid":         {"1"},
		"sectimeout":     {timeoutSeconds},
		"secidletime":    {"0"},
		"use_accountids": {"1"},
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

	response := &ChatResponse{}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return response, nil
}

func (session *Session) ChatFriendState(sid SteamID) (*ChatFriendResponse, error) {
	resp, err := session.client.Get("https://steamcommunity.com/chat/friendstate/" + strconv.FormatUint(uint64(sid.GetAccountID()), 10))
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %d", resp.StatusCode)
	}

	response := &ChatFriendResponse{}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return nil, err
	}

	return response, nil
}

func (session *Session) ChatLog(partner uint32) ([]*ChatLogMessage, error) {
	resp, err := session.client.PostForm(fmt.Sprintf("https://steamcommunity.com/chat/chatlog/%d", partner), url.Values{
		"sessionid": {session.sessionID},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	log := []*ChatLogMessage{}
	if err = json.NewDecoder(resp.Body).Decode(&log); err != nil {
		return nil, err
	}

	return log, nil
}

package steam

import (
	"encoding/json"
	"errors"
	"net/url"
)

var (
	ErrInvalidPhoneNumber = errors.New("invalid phone number specified")
)

type PhoneAPIResponse struct {
	State     string `json:"state"`
	ErrorText string `json:"errorText"`
}

func (session *Session) PrepareForSteamStore() {
	commu, _ := url.Parse("https://steamcommunity.com")
	store, _ := url.Parse("https://store.steampowered.com")

	session.client.Jar.SetCookies(store, session.client.Jar.Cookies(commu))
}

func (session *Session) ValidatePhoneNumber(number string) error {
	resp, err := session.client.Get("https://store.steampowered.com/phone/validate?phoneNumber=" + url.QueryEscape(number))
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	type Response struct {
		Success bool `json:"success"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Success {
		return ErrInvalidPhoneNumber
	}

	return nil
}

func (session *Session) AddPhoneNumber(number string) error {
	resp, err := session.client.Get("https://store.steampowered.com/phone/add_ajaxop?" + url.Values{
		"op":        {"get_phone_number"},
		"input":     {number},
		"sessionID": {session.sessionID},
		"confirmed": {"0"},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var response PhoneAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if response.State != "get_sms_code" {
		return errors.New(response.ErrorText)
	}

	return nil
}

func (session *Session) VerifyPhoneNumber(code string) error {
	resp, err := session.client.Get("https://store.steampowered.com/phone/add_ajaxop?" + url.Values{
		"op":        {"get_sms_code"},
		"input":     {code},
		"sessionID": {session.sessionID},
		"confirmed": {"0"},
	}.Encode())
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var response PhoneAPIResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if response.State != "done" {
		return errors.New(response.ErrorText)
	}

	return nil
}

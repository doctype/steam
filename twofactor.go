package steam

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"time"
)

type TwoFactorInfo struct {
	Status         uint32 `json:"status"`
	SharedSecret   string `json:"shared_secret"`
	IdentitySecret string `json:"identity_secret"`
	Secret1        string `json:"secret_1"`
	SerialNumber   uint64 `json:"serial_number,string"`
	RevocationCode string `json:"revocation_code"`
	URI            string `json:"uri"`
	ServerTime     uint64 `json:"server_time,string"`
	TokenGID       string `json:"token_gid"`
}

type FinalizeTwoFactorInfo struct {
	Status     uint32 `json:"status"`
	ServerTime uint64 `json:"server_time,string"`
}

const (
	enableTwoFactorURL   = "https://api.steampowered.com/ITwoFactorService/AddAuthenticator/v1/"
	finalizeTwoFactorURL = "https://api.steampowered.com/ITwoFactorService/FinalizeAddAuthenticator/v1/"
	disableTwoFactorURL  = "https://api.steampowered.com/ITwoFactorService/RemoveAuthenticator/v1/"
)

var ErrCannotDisable = errors.New("unable to process disable two factor request")

func (session *Session) EnableTwoFactor() (*TwoFactorInfo, error) {
	resp, err := session.client.PostForm(enableTwoFactorURL, url.Values{
		"steamid":            {session.oauth.SteamID.ToString()},
		"access_token":       {session.oauth.Token},
		"authenticator_time": {strconv.FormatInt(time.Now().Unix(), 10)},
		"authenticator_type": {"1"}, /* 1 = Valve's, 2 = thirdparty  */
		"device_identifier":  {session.deviceID},
		"sms_phone_id":       {"1"},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner *TwoFactorInfo `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

func (session *Session) FinalizeTwoFactor(authCode, mobileCode string) (*FinalizeTwoFactorInfo, error) {
	resp, err := session.client.PostForm(finalizeTwoFactorURL, url.Values{
		"steamid":            {session.oauth.SteamID.ToString()},
		"access_token":       {session.oauth.Token},
		"authenticator_time": {strconv.FormatInt(time.Now().Unix(), 10)},
		"authenticator_code": {authCode},
		"activation_code":    {mobileCode},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner *FinalizeTwoFactorInfo `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

func (session *Session) DisableTwoFactor(revocationCode string) error {
	resp, err := session.client.PostForm(disableTwoFactorURL, url.Values{
		"steamid":           {session.oauth.SteamID.ToString()},
		"access_token":      {session.oauth.Token},
		"revocation_code":   {revocationCode},
		"steamguard_scheme": {"1"},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	type Disabled struct {
		Success bool `json:"success"`
	}
	type Response struct {
		Inner *Disabled `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Inner.Success {
		return ErrCannotDisable
	}

	return nil
}

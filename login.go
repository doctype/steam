/**
  Steam Library For Go
  Copyright (C) 2016 Ahmed Samy <f.fallen45@gmail.com>
  Copyright (C) 2016 Mark Samman <mark.samman@gmail.com>

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
package steam

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

type LoginResponse struct {
	Success      bool   `json:"success"`
	PublicKeyMod string `json:"publickey_mod"`
	PublicKeyExp string `json:"publickey_exp"`
	Timestamp    string
	TokenGID     string
}

type OAuth struct {
	SteamID       SteamID `json:"steamid,string"`
	Token         string  `json:"oauth_token"`
	WGToken       string  `json:"wgtoken"`
	WGTokenSecure string  `json:"wgtoken_secure"`
	WebCookie     string  `json:"webcookie"`
}

type LoginSession struct {
	Success           bool   `json:"success"`
	LoginComplete     bool   `json:"login_complete"`
	RequiresTwoFactor bool   `json:"requires_twofactor"`
	Message           string `json:"message"`
	RedirectURI       string `json:"redirect_uri"`
	OAuthInfo         string `json:"oauth"`
}

type Community struct {
	client    *http.Client
	oauth     OAuth
	sessionID string
	apiKey    string
	deviceID  string
}

const (
	deviceIDCookieName = "steamMachineAuth"

	httpXRequestedWithValue = "com.valvesoftware.android.steam.community"
	httpUserAgentValue      = "Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
	httpAcceptValue         = "text/javascript, text/html, application/xml, text/xml, */*"
)

var (
	ErrUnableToLogin             = errors.New("unable to login")
	ErrInvalidUsername           = errors.New("invalid username")
	ErrNeedTwoFactor             = errors.New("invalid twofactor code")
	ErrMachineAuthCookieNotFound = errors.New("machine auth cookie not found")
)

func (community *Community) proceedDirectLogin(response *LoginResponse, accountName, password, sharedSecret string) error {
	n := &big.Int{}
	n.SetString(response.PublicKeyMod, 16)

	exp, err := strconv.ParseInt(response.PublicKeyExp, 16, 32)
	if err != nil {
		return err
	}

	pub := &rsa.PublicKey{N: n, E: int(exp)}
	rsaOut, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return err
	}

	var twoFactorCode string
	if sharedSecret != "" {
		if twoFactorCode, err = GenerateTwoFactorCode(sharedSecret); err != nil {
			return err
		}
	}

	params := url.Values{
		"captcha_text":      {""},
		"captchagid":        {"-1"},
		"emailauth":         {""},
		"emailsteamid":      {""},
		"password":          {base64.StdEncoding.EncodeToString(rsaOut)},
		"remember_login":    {"true"},
		"rsatimestamp":      {response.Timestamp},
		"twofactorcode":     {twoFactorCode},
		"username":          {accountName},
		"oauth_client_id":   {"DE45CD61"},
		"oauth_scope":       {"read_profile write_profile read_client write_client"},
		"loginfriendlyname": {"#login_emailauth_friendlyname_mobile"},
		"donotcache":        {strconv.FormatInt(time.Now().Unix()*1000, 10)},
	}

	req, err := http.NewRequest(http.MethodPost, "https://steamcommunity.com/login/dologin/?"+params.Encode(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-Requested-With", httpXRequestedWithValue)
	req.Header.Add("Referer", "https://steamcommunity.com/mobilelogin?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client")
	req.Header.Add("User-Agent", httpUserAgentValue)
	req.Header.Add("Accept", httpAcceptValue)

	resp, err := community.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var session LoginSession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return err
	}

	if !session.Success {
		if session.RequiresTwoFactor {
			return ErrNeedTwoFactor
		}

		return ErrUnableToLogin
	}

	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return err
	}

	sessionID := make([]byte, hex.EncodedLen(len(randomBytes)))
	hex.Encode(sessionID, randomBytes)
	community.sessionID = string(sessionID)

	url, _ := url.Parse("https://steamcommunity.com")
	cookies := community.client.Jar.Cookies(url)
	for _, cookie := range cookies {
		if cookie.Name == "mobileClient" || cookie.Name == "mobileClientVersion" {
			// remove by setting max age -1
			cookie.MaxAge = -1
		}
	}

	if sharedSecret != "" {
		for _, cookie := range cookies {
			name := cookie.Name
			if len(name) <= len(deviceIDCookieName) || name[:len(deviceIDCookieName)] != deviceIDCookieName {
				continue
			}

			shaSum := sha1.Sum([]byte(name[len(deviceIDCookieName):]))
			halfHash := shaSum[:sha1.Size/2]
			sum := make([]byte, hex.EncodedLen(len(halfHash)))
			hex.Encode(sum, halfHash)

			var deviceID bytes.Buffer
			deviceID.Grow(8 + 4 + 20)
			deviceID.WriteString("android:")
			deviceID.Write(sum[:4])
			deviceID.WriteByte('-')
			deviceID.Write(sum[4:8])
			deviceID.WriteByte('-')
			deviceID.Write(sum[8:12])
			deviceID.WriteByte('-')
			deviceID.Write(sum[12:16])
			deviceID.WriteByte('-')
			deviceID.Write(sum[16:20])
			community.deviceID = deviceID.String()
			break
		}

		if community.deviceID == "" {
			return ErrMachineAuthCookieNotFound
		}
	}

	community.client.Jar.SetCookies(
		url,
		append(cookies, &http.Cookie{
			Name:  "sessionid",
			Value: community.sessionID,
		}),
	)

	if err := json.Unmarshal([]byte(session.OAuthInfo), &community.oauth); err != nil {
		return err
	}
	return nil
}

func (community *Community) Login(accountName, password, sharedSecret string) error {
	req, err := http.NewRequest(http.MethodPost, "https://steamcommunity.com/login/getrsakey?username="+accountName, nil)
	if err != nil {
		return err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}

	req.Header.Add("X-Requested-With", httpXRequestedWithValue)
	req.Header.Add("Referer", "https://steamcommunity.com/mobilelogin?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client")
	req.Header.Add("User-Agent", httpUserAgentValue)
	req.Header.Add("Accept", httpAcceptValue)

	cookies := []*http.Cookie{
		&http.Cookie{Name: "mobileClientVersion", Value: "0 (2.1.3)"},
		&http.Cookie{Name: "mobileClient", Value: "android"},
		&http.Cookie{Name: "Steam_Language", Value: "english"},
		&http.Cookie{Name: "timezoneOffset", Value: "0,0"},
	}
	url, _ := url.Parse("https://steamcommunity.com")
	jar.SetCookies(url, cookies)

	// Construct the client
	community.client = &http.Client{Jar: jar}

	resp, err := community.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var response LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	if !response.Success {
		return ErrInvalidUsername
	}

	return community.proceedDirectLogin(&response, accountName, password, sharedSecret)
}

func (community *Community) GetSteamID() SteamID {
	return community.oauth.SteamID
}

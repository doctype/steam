package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

/* Uppercase because of JSON.  */
type LoginResponse struct {
	Success      bool
	PublicKeyMod string `json:"publickey_mod"`
	PublicKeyExp string `json:"publickey_exp"`
	Timestamp    string
	TokenGID     string
}

type TransferParams struct {
	SteamID        string
	Token          string
	Auth           string
	RememeberLogin bool `json:"remember_login"`
	WebCookie      string
	TokenSecure    string `json:"token_secure"`
}

type LoginSession struct {
	Success            bool
	LoginComplete      bool           `json:"login_complete"`
	RequiresTwoFactor  bool           `json:"requires_twofactor"`
	Message            string         `json:"message"`
	ClearPasswordField bool           `json:"clear_password_field"`
	TransferURLs       []string       `json:"transfer_urls"`
	TransferParameters TransferParams `json:"transfer_parameters"`
}

type Community struct {
	client    *http.Client
	session   LoginSession
	sessionID string
}

const (
	httpXRequestedWithValue = "com.valvesoftware.android.steam.community"
	httpUserAgentValue      = "Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
	httpAcceptValue         = "text/javascript, text/html, application/xml, text/xml, */*"
)

var (
	ErrUnableToLogin = errors.New("unable to login")
)

func (community *Community) proceedDirectLogin(response *LoginResponse, accountName, password, twoFactor string) error {
	n := &big.Int{}
	n.SetString(response.PublicKeyMod, 16)

	exp, err := strconv.ParseInt(response.PublicKeyExp, 16, 32)
	if err != nil {
		return err
	}

	itimestamp, err := strconv.ParseInt(response.Timestamp, 10, 64)
	if err != nil {
		return err
	}

	pub := &rsa.PublicKey{N: n, E: int(exp)}
	hex, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return err
	}

	b64 := base64.StdEncoding.EncodeToString(hex)
	params := fmt.Sprintf(`https://steamcommunity.com/login/dologin/?captcha_text=''&captchagid=-1&emailauth=''&emailsteamid=''&password=%s&remember_login=true&rsatimestamp=%d&twofactorcode=%s&username=%s&oauth_client_id=DE45CD61&oauth_scope=read_profile write_profile read_client write_client&loginfriendlyname=#login_emailauth_friendlyname_mobile&donotcache=%d`,
		url.QueryEscape(b64),
		itimestamp,
		twoFactor,
		accountName,
		time.Now().Unix()*1000)
	req, err := http.NewRequest(http.MethodPost, params, nil)
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
		return ErrUnableToLogin
	}

	bytes := make([]byte, 12)
	if count, err := rand.Read(bytes); count != 12 {
		return err
	}

	community.session = session
	community.sessionID = string(bytes)

	url := &url.URL{Host: "http://steamcommunity.com"}
	cookies := community.client.Jar.Cookies(url)
	for k := range cookies {
		cookie := cookies[k]
		fmt.Printf("%d: %s = %s\n", k, cookie.Name, cookie.Value)
	}

	community.client.Jar.SetCookies(
		url,
		append(cookies, &http.Cookie{
			Name:  "sessionid",
			Value: community.sessionID,
		}),
	)
	return nil
}

func (community *Community) login(accountName, password, twoFactor string) error {
	req, err := http.NewRequest(http.MethodPost, "https://steamcommunity.com/login/getrsakey?username="+accountName, nil)
	if err != nil {
		return err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}

	// Construct the client
	community.client = &http.Client{Jar: jar}

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
	jar.SetCookies(&url.URL{Host: "https://steamcommunity.com"}, cookies)

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
		return errors.New("invalid username")
	}

	return community.proceedDirectLogin(&response, accountName, password, twoFactor)
}

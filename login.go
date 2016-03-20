package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
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
	Success      bool   `json:"success"`
	PublicKeyMod string `json:"publickey_mod"`
	PublicKeyExp string `json:"publickey_exp"`
	Timestamp    string
	TokenGID     string
}

type OAuth struct {
	SteamID       string `json:"steamid"`
	Token         string `json:"oauth_token"`
	WGToken       string `json:"wgtoken"`
	WGTokenSecure string `json:"wgtoken_secure"`
	WebCookie     string `json:"webcookie"`
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
	session   LoginSession
	sessionID string
}

const (
	httpXRequestedWithValue = "com.valvesoftware.android.steam.community"
	httpUserAgentValue      = "Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
	httpAcceptValue         = "text/javascript, text/html, application/xml, text/xml, */*"
)

var (
	ErrUnableToLogin       = errors.New("unable to login")
	ErrInvalidUsername     = errors.New("invalid username")
	ErrInsufficientEntropy = errors.New("insufficient entropy")
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
	rsaOut, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return err
	}

	params := fmt.Sprintf(`https://steamcommunity.com/login/dologin/?captcha_text=''&captchagid=-1&emailauth=''&emailsteamid=''&password=%s&remember_login=true&rsatimestamp=%d&twofactorcode=%s&username=%s&oauth_client_id=DE45CD61&oauth_scope=read_profile write_profile read_client write_client&loginfriendlyname=#login_emailauth_friendlyname_mobile&donotcache=%d`,
		url.QueryEscape(base64.StdEncoding.EncodeToString(rsaOut)),
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

	randomBytes := make([]byte, 6)
	if count, err := rand.Read(randomBytes); err != nil {
		return err
	} else if count != 6 {
		return ErrInsufficientEntropy
	}

	sessionID := make([]byte, hex.EncodedLen(len(randomBytes)))
	hex.Encode(sessionID, randomBytes)

	community.session = session
	community.sessionID = string(sessionID)
	fmt.Println(session)
	fmt.Println(community.sessionID)

	url, _ := url.Parse("https://steamcommunity.com")
	cookies := community.client.Jar.Cookies(url)
	community.client.Jar.SetCookies(
		url,
		append(cookies, &http.Cookie{
			Name:  "sessionid",
			Value: community.sessionID,
		}),
	)

	var oauth OAuth
	if err := json.Unmarshal([]byte(session.OAuthInfo), &oauth); err != nil {
		return err
	}
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

	return community.proceedDirectLogin(&response, accountName, password, twoFactor)
}

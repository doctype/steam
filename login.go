package steam

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
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

type Session struct {
	client      *http.Client
	oauth       OAuth
	sessionID   string
	apiKey      string
	deviceID    string
	umqID       string
	chatMessage int
}

const (
	httpXRequestedWithValue = "com.valvesoftware.android.steam.community"
	httpUserAgentValue      = "Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30"
	httpAcceptValue         = "text/javascript, text/html, application/xml, text/xml, */*"
)

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrNeedTwoFactor   = errors.New("invalid twofactor code")
)

func (session *Session) proceedDirectLogin(response *LoginResponse, accountName, password, sharedSecret string, timeOffset time.Duration) error {
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
	if len(sharedSecret) != 0 {
		if twoFactorCode, err = GenerateTwoFactorCode(sharedSecret, time.Now().Add(timeOffset).Unix()); err != nil {
			return err
		}
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://steamcommunity.com/login/dologin/?"+url.Values{
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
		}.Encode(),
		nil,
	)
	if err != nil {
		return err
	}

	req.Header.Add("X-Requested-With", httpXRequestedWithValue)
	req.Header.Add("Referer", "https://steamcommunity.com/mobilelogin?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client")
	req.Header.Add("User-Agent", httpUserAgentValue)
	req.Header.Add("Accept", httpAcceptValue)

	resp, err := session.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return err
	}

	var loginSession LoginSession
	if err := json.NewDecoder(resp.Body).Decode(&loginSession); err != nil {
		return err
	}

	if !loginSession.Success {
		if loginSession.RequiresTwoFactor {
			return ErrNeedTwoFactor
		}

		return errors.New(loginSession.Message)
	}

	if err := json.Unmarshal([]byte(loginSession.OAuthInfo), &session.oauth); err != nil {
		return err
	}

	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return err
	}

	sessionID := make([]byte, hex.EncodedLen(len(randomBytes)))
	hex.Encode(sessionID, randomBytes)
	session.sessionID = string(sessionID)

	url, _ := url.Parse("https://steamcommunity.com")
	cookies := session.client.Jar.Cookies(url)
	for _, cookie := range cookies {
		if cookie.Name == "mobileClient" || cookie.Name == "mobileClientVersion" || cookie.Name == "steamCountry" || strings.Contains(cookie.Name, "steamMachineAuth") {
			// remove by setting max age -1
			cookie.MaxAge = -1
		}
	}

	sum := md5.Sum([]byte(password))
	session.deviceID = fmt.Sprintf(
		"android:%x-%x-%x-%x-%x",
		sum[:2], sum[2:4], sum[4:6], sum[6:8], sum[8:10],
	)

	session.client.Jar.SetCookies(
		url,
		append(cookies, &http.Cookie{
			Name:  "sessionid",
			Value: session.sessionID,
		}),
	)

	io.Copy(ioutil.Discard, resp.Body)
	return nil
}

func (session *Session) Login(accountName, password, sharedSecret string, timeOffset time.Duration) error {
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
	session.client = &http.Client{Jar: jar}

	resp, err := session.client.Do(req)
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

	io.Copy(ioutil.Discard, resp.Body)
	return session.proceedDirectLogin(&response, accountName, password, sharedSecret, timeOffset)
}

func (session *Session) GetSteamID() SteamID {
	return session.oauth.SteamID
}

func NewSessionWithAPIKey(apiKey string) *Session {
	return &Session{
		client: &http.Client{},
		apiKey: apiKey,
	}
}
func NewSession(client *http.Client, apiKey string) *Session {
	return &Session{
		client: client,
		apiKey: apiKey,
	}
}

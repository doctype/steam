package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

/* Uppercase because of JSON.  */
type LoginResponse struct {
	Success       bool
	Publickey_mod string
	Publickey_exp string
	Timestamp     string
	Token_gid     string
}

type TransferParams struct {
	SteamID         string
	Token           string
	Auth            string
	Rememeber_Login bool
	WebCookie       string
	Token_Secure    string
}

type LoginSession struct {
	Success              bool
	Login_Complete       bool
	Requires_TwoFactor   bool
	Message              string
	Clear_Password_Field bool
	Transfer_URLs        []string
	Transfer_Parameters  TransferParams
}

type Community struct {
	client    *http.Client
	session   LoginSession
	sessionId string
}

func (community *Community) proceedDirectLogin(response *LoginResponse, accountName string, password string, twoFactor string) (err error) {
	n := &big.Int{}
	n.SetString(response.Publickey_mod, 16)

	exp, err := strconv.ParseInt(response.Publickey_exp, 16, 32)
	if err != nil {
		return
	}

	itimestamp, err := strconv.ParseInt(response.Timestamp, 10, 64)
	if err != nil {
		return
	}

	pub := &rsa.PublicKey{N: n, E: int(exp)}
	hex, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return
	}

	b64 := base64.StdEncoding.EncodeToString(hex)
	params := fmt.Sprintf(`https://steamcommunity.com/login/dologin/?captcha_text=''&captchagid=-1&emailauth=''&emailsteamid=''&password=%s&remember_login=true&rsatimestamp=%d&twofactorcode=%s&username=%s&oauth_client_id=DE45CD61&oauth_scope=read_profile write_profile read_client write_client&loginfriendlyname=#login_emailauth_friendlyname_mobile&donotcache=%d`,
		url.QueryEscape(b64),
		itimestamp,
		twoFactor,
		accountName,
		time.Now().Unix()*1000)
	req, err := http.NewRequest("POST", params, nil)
	if err != nil {
		return
	}

	req.Header.Add("X-Requested-With", "com.valvesoftware.android.steam.community")
	req.Header.Add("Referer", "https://steamcommunity.com/mobilelogin?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30")
	req.Header.Add("Accept", "text/javascript, text/html, application/xml, text/xml, */*")

	resp, err := community.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var session LoginSession
	json := json.NewDecoder(resp.Body)
	err = json.Decode(&session)
	if err != nil {
		return
	}

	if !session.Success {
		return errors.New("unable to login")
	}

	fmt.Println(session)
	community.session = session
	return
}

func (community *Community) login(accountName string, password string, twoFactor string) (err error) {
	req, err := http.NewRequest("POST", "https://steamcommunity.com/login/getrsakey?username="+accountName, nil)
	if err != nil {
		return
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return
	}

	// Construct the client
	community.client = &http.Client{Jar: jar}

	req.Header.Add("X-Requested-With", "com.valvesoftware.android.steam.community")
	req.Header.Add("Referer", "https://steamcommunity.com/mobilelogin?oauth_client_id=DE45CD61&oauth_scope=read_profile%20write_profile%20read_client%20write_client")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Linux; U; Android 4.1.1; en-us; Google Nexus 4 - 4.1.1 - API 16 - 768x1280 Build/JRO03S) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30")
	req.Header.Add("Accept", "text/javascript, text/html, application/xml, text/xml, */*")

	cookies := []*http.Cookie{
		&http.Cookie{Name: "mobileClientVersion", Value: "0 (2.1.3)"},
		&http.Cookie{Name: "mobileClient", Value: "android"},
		&http.Cookie{Name: "Steam_Language", Value: "english"},
		&http.Cookie{Name: "timezoneOffset", Value: "0,0"},
	}
	jar.SetCookies(&url.URL{Host: "https://steamcommunity.com"}, cookies)

	resp, err := community.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var response LoginResponse
	json := json.NewDecoder(resp.Body)
	err = json.Decode(&response)
	if err != nil {
		return
	}

	if !response.Success {
		return errors.New("invalid username")
	}

	return community.proceedDirectLogin(&response, accountName, password, twoFactor)
}

package steam

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
)

const (
	apiKeyURL         = "https://steamcommunity.com/dev/apikey"
	apiKeyRegisterURL = "https://steamcommunity.com/dev/registerkey"
	apiKeyRevokeURL   = "https://steamcommunity.com/dev/revokekey"

	accessDeniedPattern = "<h2>Access Denied</h2>"
)

var (
	keyRegExp = regexp.MustCompile("<p>Key: ([0-9A-F]+)</p>")

	ErrCannotRegisterKey = errors.New("unable to register API key")
	ErrCannotRevokeKey   = errors.New("unable to revoke API key")
	ErrAccessDenied      = errors.New("access is denied")
	ErrKeyNotFound       = errors.New("key not found")
)

func (session *Session) parseKey(resp *http.Response) (string, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if m, err := regexp.Match(accessDeniedPattern, body); err != nil {
		return "", err
	} else if m {
		return "", ErrAccessDenied
	}

	submatch := keyRegExp.FindStringSubmatch(string(body))
	if len(submatch) != 2 {
		return "", ErrKeyNotFound
	}

	session.apiKey = submatch[1]
	return submatch[1], nil
}

func (session *Session) RegisterWebAPIKey(domain string) (string, error) {
	resp, err := session.client.PostForm(apiKeyRegisterURL, url.Values{
		"domain":       {domain},
		"agreeToTerms": {"agreed"},
		"sessionid":    {session.sessionID},
		"Submit":       {"Register"},
	})
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", ErrCannotRegisterKey
	}

	return session.parseKey(resp)
}

func (session *Session) GetWebAPIKey() (string, error) {
	resp, err := session.client.Get(apiKeyURL)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", err
	}

	return session.parseKey(resp)
}

func (session *Session) RevokeWebAPIKey() error {
	resp, err := session.client.PostForm(apiKeyRevokeURL, url.Values{
		"Revoke":    {"Revoke My Steam Web API Key"},
		"sessionid": {session.sessionID},
	})
	if resp != nil {
		resp.Body.Close()
	}

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return ErrCannotRevokeKey
	}

	return nil
}

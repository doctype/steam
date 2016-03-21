package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
)

const (
	apiKeyURL  = "https://steamcommunity.com/dev/apikey"
	apiCallURL = "https://api.steampowered.com/IEconService/"

	accessDeniedPattern = "<h2>Access Denied</h2>"
)

var (
	keyRegExp = regexp.MustCompile("<p>Key: ([0-9A-F]+)</p>")

	ErrAccessDenied = errors.New("access is denied")
	ErrKeyNotFound  = errors.New("key not found")
)

func (community *Community) getWebAPIKey() (string, error) {
	req, err := http.NewRequest(http.MethodGet, apiKeyURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := community.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return "", err
	}

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
	if len(submatch) == 0 {
		return "", ErrKeyNotFound
	}

	community.apiKey = submatch[1]
	return submatch[1], nil
}

func (community *Community) MakeAPICall(method string, request string, values *url.Values) (body []byte, err error) {
	req, err := http.NewRequest(method, apiCallURL+request+"/v1/?"+values.Encode(), nil)
	if err != nil {
		return
	}

	resp, err := community.client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return
	}

	eresult := resp.Header.Get("x-eresult")
	if eresult != "1" {
		return body, errors.New(eresult)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return body, nil
}

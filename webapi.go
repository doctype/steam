package main

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

func (community *Community) getWebApiKey() (string, error) {
	req, err := http.NewRequest("POST", "https://steamcommunity.com/dev/apikey", nil)
	if err != nil {
		return "", err
	}

	resp, err := community.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fmt.Println(resp)
	body := make([]byte, resp.ContentLength)
	n, err := resp.Body.Read(body)
	if err != nil || n != int(resp.ContentLength) {
		return "", err
	}

	m, err := regexp.MatchString("<h2>Access Denied</h2>", string(body))
	if err != nil {
		return "", err
	}

	if m {
		return "", errors.New("access is denied")
	}

	re, err := regexp.Compile("<p>Key: ([0-9A-F]+)</p>")
	if err != nil {
		return "", err
	}

	fmt.Println(re)
	return re.FindStringSubmatch("Key")[0], nil
}

package steam

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"time"
)

const (
	chars    = "23456789BCDFGHJKMNPQRTVWXY"
	charsLen = uint32(len(chars))
)

type ServerTimeTip struct {
	Time                              int64  `json:"server_time,string"`
	SkewToleranceSeconds              uint32 `json:"skew_tolerance_seconds,string"`
	LargeTimeJink                     uint32 `json:"large_time_jink,string"`
	ProbeFrequencySeconds             uint32 `json:"probe_frequency_seconds"`
	AdjustedTimeProbeFrequencySeconds uint32 `json:"adjusted_time_probe_frequency_seconds"`
	HintProbeFrequencySeconds         uint32 `json:"hint_probe_frequency_seconds"`
	SyncTimeout                       uint32 `json:"sync_timeout"`
	TryAgainSeconds                   uint32 `json:"try_again_seconds"`
	MaxAttempts                       uint32 `json:"max_attempts"`
}

func GenerateTwoFactorCode(sharedSecret string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(sharedSecret)
	if err != nil {
		return "", err
	}

	ful := make([]byte, 8)
	binary.BigEndian.PutUint32(ful[4:], uint32(time.Now().Unix()/30))

	hmac := hmac.New(sha1.New, data)
	hmac.Write(ful)

	sum := hmac.Sum(nil)
	start := sum[19] & 0x0F
	slice := binary.BigEndian.Uint32(sum[start:start+4]) & 0x7FFFFFFF

	buf := make([]byte, 5)
	for i := 0; i < 5; i++ {
		buf[i] = chars[slice%charsLen]
		slice /= charsLen
	}
	return string(buf), nil
}

func GenerateConfirmationCode(identitySecret, tag string, current int64) (string, error) {
	data, err := base64.StdEncoding.DecodeString(identitySecret)
	if err != nil {
		return "", err
	}

	ful := make([]byte, 8+len(tag))
	binary.BigEndian.PutUint32(ful[4:], uint32(current))
	copy(ful[8:], tag)

	hmac := hmac.New(sha1.New, data)
	hmac.Write(ful)

	return base64.StdEncoding.EncodeToString(hmac.Sum(nil)), nil
}

func GetTimeTip() (*ServerTimeTip, error) {
	resp, err := http.Post("https://api.steampowered.com/ITwoFactorService/QueryTime/v1/", "application/x-www-form-urlencoded", nil)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	type Response struct {
		Inner *ServerTimeTip `json:"response"`
	}

	var response Response
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Inner, nil
}

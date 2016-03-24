package steam

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"time"
)

const (
	chars    = "23456789BCDFGHJKMNPQRTVWXY"
	charsLen = uint32(len(chars))
)

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

func GenerateConfirmationCode(identitySecret, tag string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(identitySecret)
	if err != nil {
		return "", err
	}

	ful := make([]byte, 8+len(tag))
	binary.BigEndian.PutUint32(ful[4:], uint32(time.Now().Unix()))
	copy(ful[8:], tag)

	hmac := hmac.New(sha1.New, data)
	hmac.Write(ful)

	return base64.StdEncoding.EncodeToString(hmac.Sum(nil)), nil
}

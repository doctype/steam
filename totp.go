package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"time"
)

func GenerateTwoFactorCode(shared_secret string) string {
	data, err := base64.StdEncoding.DecodeString(shared_secret)
	if err != nil {
		return ""
	}

	ful := make([]byte, 8)
	binary.BigEndian.PutUint32(ful, 0)
	binary.BigEndian.PutUint32(ful[4:], uint32(time.Now().Unix()/30))

	hmac := hmac.New(sha1.New, data)
	hmac.Write(ful)

	sum := hmac.Sum(nil)
	start := sum[19] & 0x0F
	slice := binary.BigEndian.Uint32(sum[start:start+4]) & 0x7FFFFFFF
	chars := "23456789BCDFGHJKMNPQRTVWXY"
	len := uint32(len(chars))

	buf := make([]byte, 5)
	for i := 0; i < 5; i++ {
		buf[i] = chars[slice%len]
		slice /= len
	}

	return string(buf)
}

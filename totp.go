/**
  Steam Library For Go
  Copyright (C) 2016 Ahmed Samy <f.fallen45@gmail.com>
  Copyright (C) 2016 Mark Samman <mark.samman@gmail.com>

  This library is free software; you can redistribute it and/or
  modify it under the terms of the GNU Lesser General Public
  License as published by the Free Software Foundation; either
  version 2.1 of the License, or (at your option) any later version.

  This library is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
  Lesser General Public License for more details.

  You should have received a copy of the GNU Lesser General Public
  License along with this library; if not, write to the Free Software
  Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
*/
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

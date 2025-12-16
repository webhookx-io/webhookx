package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
)

var (
	ErrUnsupportedHashMethod     = errors.New("unsupported hash method")
	ErrUnsupportedEncodingMethod = errors.New("unsupported encoding method")
)

var (
	factories = map[string]func() hash.Hash{
		"md5":     md5.New,
		"sha-1":   sha1.New,
		"sha-256": sha256.New,
		"sha-512": sha512.New,
	}
)

func Hmac(hash string, key []byte, data []byte) []byte {
	fn, exist := factories[hash]
	if !exist {
		panic(fmt.Errorf("%w: %s", ErrUnsupportedHashMethod, hash))
	}
	h := hmac.New(fn, []byte(key))
	h.Write([]byte(data))
	return h.Sum(nil)
}

func HmacEncode(hash string, key []byte, data []byte, encoding string) string {
	b := Hmac(hash, key, data)
	switch encoding {
	case "hex":
		return hex.EncodeToString(b)
	case "base64":
		return base64.StdEncoding.EncodeToString(b)
	case "base64url":
		return base64.RawURLEncoding.EncodeToString(b)
	default:
		panic(fmt.Errorf("%w: %s", ErrUnsupportedEncodingMethod, encoding))
	}
}

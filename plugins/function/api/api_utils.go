package api

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash"
)

type UtilsAPI struct{}

func NewUtilsAPI() *UtilsAPI {
	return &UtilsAPI{}
}

func (api *UtilsAPI) Hmac(algorithm string, key string, data string) []byte {
	var fn func() hash.Hash
	switch algorithm {
	case "SHA-1":
		fn = sha1.New
	case "SHA-256":
		fn = sha256.New
	case "SHA-512":
		fn = sha512.New
	case "MD5":
		fn = md5.New
	default:
		panic(errors.New("unknown algorithm: " + algorithm))
	}
	mac := hmac.New(fn, []byte(key))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func (api *UtilsAPI) Encode(name string, data []byte) string {
	switch name {
	case "hex":
		return hex.EncodeToString(data)
	case "base64":
		return base64.StdEncoding.EncodeToString(data)
	case "base64url":
		return base64.RawURLEncoding.EncodeToString(data)
	default:
		panic(errors.New("unknown encode type: " + name))
	}
}

func (api *UtilsAPI) DigestEqual(a string, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

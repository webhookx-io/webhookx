package hmac_auth

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

var (
	ErrInvalidHashMethod     = errors.New("invalid hash method")
	ErrInvalidEncodingMethod = errors.New("invalid encoding method")
)

var hashes = map[string]func() hash.Hash{
	"md5":     md5.New,
	"sha-1":   sha1.New,
	"sha-256": sha256.New,
	"sha-512": sha512.New,
}

func Hmac(algorithm string, key string, data string) []byte {
	fn, ok := hashes[algorithm]
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrInvalidHashMethod, algorithm))
	}
	h := hmac.New(fn, []byte(key))
	h.Write([]byte(data))
	return h.Sum(nil)
}

func encode(encoding string, data []byte) string {
	switch encoding {
	case "hex":
		return hex.EncodeToString(data)
	case "base64":
		return base64.StdEncoding.EncodeToString(data)
	case "base64url":
		return base64.RawURLEncoding.EncodeToString(data)
	default:
		panic(fmt.Errorf("%w: %s", ErrInvalidEncodingMethod, encoding))
	}
}

type Config struct {
	Hash            string `json:"hash"`
	Encoding        string `json:"encoding"`
	SignatureHeader string `json:"signature_header"`
	Secret          string `json:"secret"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("HmacAuthPluginConfiguration")
}

type HmacAuthPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *HmacAuthPlugin) Name() string {
	return "hmac-auth"
}

func (p *HmacAuthPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
	matched := false
	signature := inbound.Request.Header.Get(p.Config.SignatureHeader)
	if len(signature) > 0 {
		bytes := Hmac(p.Config.Hash, p.Config.Secret, string(inbound.RawBody))
		expectedSignature := encode(p.Config.Encoding, bytes)
		matched = subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
	}

	if !matched {
		response.JSON(inbound.Response, 401, `{"message":"Unauthorized"}`)
		result.Terminated = true
	}
	result.Payload = inbound.RawBody

	return
}

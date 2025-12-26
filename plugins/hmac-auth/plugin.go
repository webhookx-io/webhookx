package hmac_auth

import (
	"context"
	"crypto/subtle"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
)

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

func (p *HmacAuthPlugin) Priority() int {
	return 107
}

func (p *HmacAuthPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
	cfg := p.Config
	matched := false
	signature := inbound.Request.Header.Get(p.Config.SignatureHeader)
	if len(signature) > 0 {
		expectedSignature := utils.HmacEncode(cfg.Hash, []byte(cfg.Secret), inbound.RawBody, cfg.Encoding)
		matched = subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
	}

	if !matched {
		response.JSON(inbound.Response, 401, `{"message":"Unauthorized"}`)
		result.Terminated = true
	}
	result.Payload = inbound.RawBody

	return
}

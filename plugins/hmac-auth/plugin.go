package hmac_auth

import (
	"crypto/subtle"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
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

func (p *HmacAuthPlugin) ExecuteInbound(c *plugin.Context) error {
	cfg := p.Config
	matched := false
	signature := c.Request.Header.Get(p.Config.SignatureHeader)
	if len(signature) > 0 {
		expectedSignature := utils.HmacEncode(cfg.Hash, []byte(cfg.Secret), c.GetRequestBody(), cfg.Encoding)
		matched = subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
	}

	if !matched {
		c.JSON(401, `{"message":"Unauthorized"}`)
	}

	return nil
}

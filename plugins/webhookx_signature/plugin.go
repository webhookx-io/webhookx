package webhookx_signature

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
)

type Config struct {
	SigningSecret string `json:"signing_secret"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("WebhookxSignaturePluginConfiguration")
}

type SignaturePlugin struct {
	plugin.BasePlugin[Config]

	ts time.Time // used in testing
}

func (p *SignaturePlugin) Name() string {
	return "webhookx-signature"
}

// TODO
func (p *SignaturePlugin) ValidateConfig(config map[string]interface{}) error {
	if _, ok := config["signing_secret"]; !ok {
		config["signing_secret"] = utils.RandomString(32)
	}

	return p.BasePlugin.ValidateConfig(config)
}

func computeSignature(ts time.Time, payload []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return mac.Sum(nil)
}

func (p *SignaturePlugin) ExecuteOutbound(ctx context.Context, outbound *plugin.Outbound) error {
	ts := p.ts
	if ts.IsZero() {
		ts = time.Now()
	}
	signature := computeSignature(ts, []byte(outbound.Payload), p.Config.SigningSecret)
	outbound.Headers["webhookx-signature"] = "v1=" + hex.EncodeToString(signature)
	outbound.Headers["webhookx-timestamp"] = strconv.FormatInt(ts.Unix(), 10)
	return nil
}

package webhookx_signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"strconv"
	"time"
)

type Config struct {
	SigningSecret string `json:"signing_secret" validate:"required"`
}

type SignaturePlugin struct {
	plugin.BasePlugin[Config]

	ts time.Time // used in testing
}

func New(config []byte) (plugin.Plugin, error) {
	p := &SignaturePlugin{}
	p.Name = "webhookx-signature"

	p.Config.SigningSecret = utils.RandomString(32)

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *SignaturePlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func computeSignature(ts time.Time, payload []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return mac.Sum(nil)
}

func (p *SignaturePlugin) ExecuteOutbound(outbound *plugin.Outbound, _ *plugin.Context) error {
	ts := p.ts
	if ts.IsZero() {
		ts = time.Now()
	}
	signature := computeSignature(ts, []byte(outbound.Payload), p.Config.SigningSecret)
	outbound.Headers["webhookx-signature"] = "v1=" + hex.EncodeToString(signature)
	outbound.Headers["webhookx-timestamp"] = strconv.FormatInt(ts.Unix(), 10)
	return nil
}

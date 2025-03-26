package webhookx_signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
	"github.com/webhookx-io/webhookx/utils"
	"strconv"
	"time"
)

type Config struct {
	SigningSecret string `json:"signing_secret" validate:"required"`
}

func (cfg *Config) Validate() error {
	return utils.Validate(cfg)
}

func (cfg *Config) ProcessDefault() {
	if cfg.SigningSecret == "" {
		cfg.SigningSecret = utils.RandomString(32)
	}
}

type SignaturePlugin struct {
	types.BasePlugin

	cfg Config

	ts time.Time // used in testing
}

func New() types.Plugin {
	plugin := &SignaturePlugin{}
	plugin.Name = "webhookx-signature"
	return plugin
}

func computeSignature(ts time.Time, payload []byte, secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(payload)
	return mac.Sum(nil)
}

func (p *SignaturePlugin) Execute(req *types.Request, context *types.Context) error {
	ts := p.ts
	if ts.IsZero() {
		ts = time.Now()
	}
	signature := computeSignature(ts, []byte(req.Payload), p.cfg.SigningSecret)
	req.Headers["webhookx-signature"] = "v1=" + hex.EncodeToString(signature)
	req.Headers["webhookx-timestamp"] = strconv.FormatInt(ts.Unix(), 10)
	return nil
}

func (p *SignaturePlugin) Config() types.PluginConfig {
	return &p.cfg
}

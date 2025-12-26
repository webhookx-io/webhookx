package verifier

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/webhookx-io/webhookx/utils"
)

type HmacConfig struct {
	Hash                   string
	Encoding               string
	SignatureHeader        string
	TimestampHeader        string
	SignaturePostProcessor SignaturePostProcessor
	MessagePostProcessor   MessagePostProcessor
}

type SignaturePostProcessor func(r *http.Request, signature string) string

type MessagePostProcessor func(r *http.Request, message string) string

type HmacOption func(*HmacConfig)

func WithSignaturePostProcessor(fn SignaturePostProcessor) HmacOption {
	return func(cfg *HmacConfig) {
		cfg.SignaturePostProcessor = fn
	}
}

func WithMessagePostProcessor(fn MessagePostProcessor) HmacOption {
	return func(cfg *HmacConfig) {
		cfg.MessagePostProcessor = fn
	}
}

func WithTimestampHeader(timestampHeader string) HmacOption {
	return func(cfg *HmacConfig) {
		cfg.TimestampHeader = timestampHeader
	}
}

var (
	SignatureSplitPostProcessor = func(sep string, n int) func(r *http.Request, signature string) string {
		return func(r *http.Request, signature string) string {
			parts := strings.SplitN(signature, sep, n)
			if len(parts) == n {
				return parts[n-1]
			}
			return signature
		}
	}
)

type HmacVerifier struct {
	cfg *HmacConfig
}

func NewHmacVerifier(hash, encoding, signatureHeader string, options ...HmacOption) *HmacVerifier {
	cfg := &HmacConfig{
		Hash:            hash,
		Encoding:        encoding,
		SignatureHeader: signatureHeader,
	}
	for _, option := range options {
		option(cfg)
	}
	return &HmacVerifier{
		cfg: cfg,
	}
}

type HmacVerifyConfig struct {
	Secret          string `json:"secret"`
	ToleranceWindow int64  `json:"tolerance_window"`
}

func (v *HmacVerifier) decodeConfig(config map[string]interface{}) (*HmacVerifyConfig, error) {
	cfg := HmacVerifyConfig{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &cfg,
	})
	if err != nil {
		return nil, err
	}
	err = decoder.Decode(config)
	return &cfg, err
}

func (v *HmacVerifier) Verify(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
	cfg := v.cfg
	r := req.R

	verifyConfig, err := v.decodeConfig(config)
	if err != nil {
		return nil, err
	}

	signature := r.Header.Get(cfg.SignatureHeader)
	if signature == "" {
		return &Result{Verified: false}, nil
	}
	if cfg.SignaturePostProcessor != nil {
		signature = cfg.SignaturePostProcessor(r, signature)
	}

	if cfg.TimestampHeader != "" && verifyConfig.ToleranceWindow > 0 {
		t := r.Header.Get(cfg.TimestampHeader)
		if t == "" {
			return &Result{Verified: false}, nil
		}
		timestamp, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil, err
		}
		if time.Now().Unix()-timestamp > verifyConfig.ToleranceWindow {
			return &Result{Verified: false}, nil
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	message := string(body)
	if cfg.MessagePostProcessor != nil {
		message = cfg.MessagePostProcessor(r, message)
	}

	expectedSignature := utils.HmacEncode(cfg.Hash, []byte(verifyConfig.Secret), []byte(message), cfg.Encoding)

	res := &Result{}
	if timingSafeEqual(signature, expectedSignature) {
		res.Verified = true
	}
	return res, nil
}

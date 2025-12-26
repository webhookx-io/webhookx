package verifier

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/webhookx-io/webhookx/utils"
)

// StandardWebhooksVerifier is standard-webhooks implementation
type StandardWebhooksVerifier struct {
}

func NewStandardWebhooksVerifier() *StandardWebhooksVerifier {
	return &StandardWebhooksVerifier{}
}

func parseSecret(secret string) ([]byte, error) {
	secret = strings.TrimPrefix(secret, "whsec_")
	return base64.StdEncoding.DecodeString(secret)
}

type StandardWebhooksVerifyConfig struct {
	Secret          string `json:"secret"`
	ToleranceWindow int64  `json:"tolerance_window"`
}

func (v *StandardWebhooksVerifier) decodeConfig(config map[string]interface{}) (*StandardWebhooksVerifyConfig, error) {
	cfg := StandardWebhooksVerifyConfig{}
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

func (v *StandardWebhooksVerifier) Verify(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
	verifyConfig, err := v.decodeConfig(config)
	if err != nil {
		return nil, err
	}

	secret, err := parseSecret(verifyConfig.Secret)
	if err != nil {
		return nil, err
	}
	r := req.R

	requestId := r.Header.Get("webhook-id")
	requestTimestamp := r.Header.Get("webhook-timestamp")
	requestSignature := r.Header.Get("webhook-signature")

	if requestId == "" || requestSignature == "" || requestTimestamp == "" {
		return &Result{Verified: false}, nil
	}

	if verifyConfig.ToleranceWindow > 0 {
		timestamp, err := strconv.ParseInt(requestTimestamp, 10, 64)
		if err != nil {
			return nil, err
		}
		if time.Now().Unix()-timestamp > verifyConfig.ToleranceWindow {
			return &Result{Verified: false}, nil
		}
	}

	payload, err := io.ReadAll(req.R.Body)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("%s.%s.%s", requestId, requestTimestamp, payload)
	expectedSignature := utils.HmacEncode("sha-256", secret, []byte(message), "base64")

	res := &Result{}

	signatures := strings.Split(requestSignature, " ")
	for _, versionedSignature := range signatures {
		parts := strings.Split(versionedSignature, ",")
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		if version != "v1" {
			continue
		}

		signature := parts[1]
		if timingSafeEqual(signature, expectedSignature) {
			res.Verified = true
			break
		}
	}

	return res, nil
}

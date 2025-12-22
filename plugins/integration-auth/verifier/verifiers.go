package verifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v84/webhook"
)

var (
	registry = map[string]Verifier{}
)

func LoadVerifier(provider string) (Verifier, bool) {
	verifier, exist := registry[provider]
	return verifier, exist
}

func init() {
	registry["github"] = NewHmacVerifier("sha-256", "hex", "X-Hub-Signature-256",
		WithSignaturePostProcessor(SignatureSplitPostProcessor("=", 2)))

	registry["slack"] = NewHmacVerifier("sha-256", "hex", "X-Slack-Signature",
		WithSignaturePostProcessor(SignatureSplitPostProcessor("=", 2)),
		WithMessagePostProcessor(func(r *http.Request, message string) string {
			return "v0:" + r.Header.Get("X-Slack-Request-Timestamp") + ":" + message
		}),
		WithTimestampHeader("X-Slack-Request-Timestamp"))

	registry["stripe"] = VerifyFunc(func(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
		secret := config["secret"].(string)
		tolerance := int(config["tolerance_window"].(float64))

		var signature = req.R.Header.Get("Stripe-Signature")
		var err error

		payload, err := io.ReadAll(req.R.Body)
		if err != nil {
			return nil, err
		}
		if tolerance > 0 {
			err = webhook.ValidatePayloadWithTolerance(payload, signature, secret,
				time.Second*time.Duration(tolerance))
		} else {
			// ignore tolerance
			err = webhook.ValidatePayloadIgnoringTolerance(payload, signature, secret)

		}
		res := &Result{}
		res.Verified = err == nil
		return res, nil
	})

	registry["gitlab"] = VerifyFunc(func(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
		secret := config["secret"].(string)
		var token = req.R.Header.Get("X-Gitlab-Token")
		res := &Result{}
		if timingSafeEqual(token, secret) {
			res.Verified = true
		}
		return res, nil
	})

	registry["zendesk"] = NewHmacVerifier("sha-256", "base64", "X-Zendesk-Webhook-Signature",
		WithMessagePostProcessor(func(r *http.Request, message string) string {
			return r.Header.Get("X-Zendesk-Webhook-Signature-Timestamp") + message
		}),
		WithTimestampHeader("X-Zendesk-Webhook-Signature-Timestamp"))

	registry["openai"] = NewStandardWebhooksVerifier()

	registry["okta"] = VerifyFunc(func(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
		r := req.R

		if r.Method == "GET" {
			response, _ := json.Marshal(map[string]string{
				"verification": r.Header.Get("X-Okta-Verification-Challenge"),
			})
			res := &Result{
				Verified: true,
				Response: &Response{
					StatusCode: http.StatusOK,
					Headers: map[string]string{
						"Content-Type": "application/json; charset=utf-8",
					},
					Body: response,
				},
			}
			return res, nil
		}

		authHeader := config["authentication_field"].(string)
		authSecret := config["authentication_secret"].(string)
		res := &Result{}
		if authHeader == "" || timingSafeEqual(authSecret, r.Header.Get(authHeader)) {
			res.Verified = true
		}
		return res, nil
	})
}

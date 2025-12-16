package verifier

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v84/webhook"
	"github.com/webhookx-io/webhookx/utils"
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

	registry["x"] = &X{
		verifier: NewHmacVerifier("sha-256", "hex", "X-Twitter-Webhooks-Signature",
			WithSignaturePostProcessor(SignatureSplitPostProcessor("=", 2))), // todo: ???
	}

	registry["gitlab"] = VerifyFunc(func(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
		secret := config["secret"].(string)
		var token = req.R.Header.Get("X-Gitlab-Token")
		res := &Result{}
		if timingSafeEqual(token, secret) {
			res.Verified = true
		}
		return res, nil
	})

	//registry["discord"] = VerifyFunc(func(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
	//	r := req.R
	//	secret := config["secret"].(string)
	//
	//	requestSignature := r.Header.Get("X-Signature-Ed25519")
	//	requestTimestamp := r.Header.Get("X-Signature-Timestamp")
	//
	//	sig, err := hex.DecodeString(requestSignature)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	pubKey, err := hex.DecodeString(secret)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	payload, err := io.ReadAll(req.R.Body)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	message := requestTimestamp + string(payload)
	//
	//	res := &Result{}
	//	res.Verified = ed25519.Verify(pubKey, []byte(message), sig)
	//	return res, nil
	//})

	registry["zendesk"] = NewHmacVerifier("sha-256", "base64", "X-Zendesk-Webhook-Signature",
		WithMessagePostProcessor(func(r *http.Request, message string) string {
			return r.Header.Get("X-Zendesk-Webhook-Signature-Timestamp") + message
		}),
		WithTimestampHeader("X-Zendesk-Webhook-Signature-Timestamp"))

	registry["openai"] = NewStandardWebhooksVerifier()

	registry["okta"] = VerifyFunc(func(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
		r := req.R

		if r.Method == "GET" {
			response := `{"verification": "` + r.Header.Get("X-Okta-Verification-Challenge") + `"}`
			res := &Result{
				Verified: true,
				Response: &Response{
					StatusCode: http.StatusOK,
					Headers: map[string]string{
						"Content-Type": "application/json; charset=utf-8",
					},
					Body: []byte(response),
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

type X struct {
	verifier *HmacVerifier
}

func (x *X) Verify(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
	// https://docs.x.com/x-api/enterprise-gnip-2.0/fundamentals/account-activity#optional-signature-header-validation
	if req.R.Method == "GET" {
		// https://docs.x.com/x-api/enterprise-gnip-2.0/fundamentals/account-activity#challenge-response-checks
		// The GET hash is calculated from the query parameter string crc_token=token&nonce=nonce.
		// TODO: validate request that is actually come from X
		token := config["secret"].(string)
		t := req.R.URL.Query().Get("crc_token")
		signature := utils.HmacEncode("sha-256", []byte(token), []byte(t), "base64")
		response := `{"response_token": "sha256=` + signature + `"}`
		res := &Result{
			Verified: true,
			Response: &Response{
				StatusCode: http.StatusOK,
				Headers: map[string]string{
					"Content-Type": "application/json; charset=utf-8",
				},
				Body: []byte(response),
			},
		}
		return res, nil
	}

	return x.verifier.Verify(ctx, req, config)
}

//type Twilio struct{}
//
//func (t *Twilio) Verify(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
//	secret := config["secret"].(string)
//	r := req.R
//
//	signature := r.Header.Get("X-Twilio-Signature")
//	validator := client.NewRequestValidator(secret)
//	url := r.URL.String() // todo  fullURL := "https://" + r.Host + r.URL.RequestURI()
//
//	ct := r.Header.Get("Content-Type")
//	var validated bool
//	if strings.Contains(strings.ToLower(ct), "application/json") {
//		payload, err := io.ReadAll(r.Body)
//		if err != nil {
//			return nil, err
//		}
//		validated = validator.ValidateBody(url, payload, signature)
//	} else {
//		if err := r.ParseForm(); err != nil {
//			return nil, err
//		}
//		params := make(map[string]string)
//		for k := range r.PostForm {
//			params[k] = r.PostForm.Get(k)
//		}
//		validated = validator.Validate(url, params, signature)
//	}
//
//	res := &Result{}
//	res.Verified = validated
//	return res, nil
//}
//
//func (t *Twilio) getURL(r *http.Request) string {
//	proto := r.Header.Get("X-Forwarded-Proto")
//	host := r.Header.Get("X-Forwarded-Host")
//	port := r.Header.Get("X-Forwarded-Port")
//	if proto == "" || host == "" || port == "" {
//		return r.URL.String()
//	}
//	return fmt.Sprintf("%s://%s:%s%s", proto, host, port, r.URL.Path)
//}

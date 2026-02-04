package plugins_test

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	hmac_auth "github.com/webhookx-io/webhookx/plugins/hmac-auth"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("hmac-auth", Ordered, func() {
	Context("", func() {
		var proxyClient *resty.Client
		var app *app.Application
		var payload = `{"event_type": "foo.bar","data": {"key": "value"}}`

		var signature []byte

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources: []*entities.Source{
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/sha256-hex"
					o.Plugins = []*entities.Plugin{factory.Plugin("hmac-auth",
						factory.WithPluginConfig(hmac_auth.Config{
							Hash:            "sha-256",
							Encoding:        "hex",
							SignatureHeader: "X-Signature",
							Secret:          "mykey",
						})),
					}
				}),
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/sha256-base64"
					o.Plugins = []*entities.Plugin{factory.Plugin("hmac-auth",
						factory.WithPluginConfig(hmac_auth.Config{
							Hash:            "sha-256",
							Encoding:        "base64",
							SignatureHeader: "X-Signature",
							Secret:          "mykey",
						})),
					}
				}),
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/sha256-base64url"
					o.Plugins = []*entities.Plugin{factory.Plugin("hmac-auth",
						factory.WithPluginConfig(hmac_auth.Config{
							Hash:            "sha-256",
							Encoding:        "base64url",
							SignatureHeader: "X-Signature",
							Secret:          "mykey",
						})),
					}
				}),
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/md5-hex"
					o.Plugins = []*entities.Plugin{factory.Plugin("hmac-auth",
						factory.WithPluginConfig(hmac_auth.Config{
							Hash:            "md5",
							Encoding:        "hex",
							SignatureHeader: "X-Signature",
							Secret:          "mykey",
						})),
					}
				}),
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/sha1-hex"
					o.Plugins = []*entities.Plugin{factory.Plugin("hmac-auth",
						factory.WithPluginConfig(hmac_auth.Config{
							Hash:            "sha-1",
							Encoding:        "hex",
							SignatureHeader: "X-Signature",
							Secret:          "mykey",
						})),
					}
				}),
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/sha512-hex"
					o.Plugins = []*entities.Plugin{factory.Plugin("hmac-auth",
						factory.WithPluginConfig(hmac_auth.Config{
							Hash:            "sha-512",
							Encoding:        "hex",
							SignatureHeader: "X-Signature",
							Secret:          "mykey",
						})),
					}
				}),
			},
		}
		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(nil))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)

			h := hmac.New(sha256.New, []byte("mykey"))
			h.Write([]byte(payload))
			signature = h.Sum(nil)
		})

		AfterAll(func() {
			app.Stop()
		})

		Context("sha256", func() {
			It("should pass when passing right signature", func() {
				paths := []string{"/sha256-hex", "/sha256-base64", "/sha256-base64url"}
				signatures := []string{
					hex.EncodeToString(signature),
					base64.StdEncoding.EncodeToString(signature),
					base64.RawURLEncoding.EncodeToString(signature),
				}

				for i, path := range paths {
					resp, err := proxyClient.R().
						SetBody(payload).
						SetHeader("X-Signature", signatures[i]).
						Post(path)
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 200, resp.StatusCode())
				}
			})

			It("should deny when missing signature", func() {
				resp, err := proxyClient.R().
					SetBody(payload).
					Post("/sha256-hex")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 401, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
			})

			It("should deny when passing wrong signature", func() {
				resp, err := proxyClient.R().
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					SetHeader("X-Signature", "wrong").
					Post("/sha256-hex")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 401, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
			})
		})

		Context("md5", func() {
			It("should pass when passing right signature", func() {
				h := hmac.New(md5.New, []byte("mykey"))
				h.Write([]byte(payload))
				signature := h.Sum(nil)
				resp, err := proxyClient.R().
					SetBody(payload).
					SetHeader("X-Signature", hex.EncodeToString(signature)).
					Post("/md5-hex")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
			})
		})

		Context("sha1", func() {
			It("should pass when passing right signature", func() {
				h := hmac.New(sha1.New, []byte("mykey"))
				h.Write([]byte(payload))
				signature := h.Sum(nil)
				resp, err := proxyClient.R().
					SetBody(payload).
					SetHeader("X-Signature", hex.EncodeToString(signature)).
					Post("/sha1-hex")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
			})
		})

		Context("sha512", func() {
			It("should pass when passing right signature", func() {
				h := hmac.New(sha512.New, []byte("mykey"))
				h.Write([]byte(payload))
				signature := h.Sum(nil)
				resp, err := proxyClient.R().
					SetBody(payload).
					SetHeader("X-Signature", hex.EncodeToString(signature)).
					Post("/sha512-hex")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
			})
		})
	})
})

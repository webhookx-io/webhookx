package plugins_test

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	integration_auth "github.com/webhookx-io/webhookx/plugins/integration-auth"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("integration-auth", Ordered, func() {

	Context("github", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "github",
					ProviderConfig: map[string]interface{}{
						"secret": "test-github-secret",
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should succeed", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Hub-Signature-256", "sha256=564a9a043559ae76eb13103f9b25f08422a5c1a17daeb75d9b929caa35291f68").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail when missing signature header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when sending a signature that doesn't match", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Hub-Signature-256", "sha256=80522b11f759e2ea6dcc63f78d20675f017030a5d417bd2daf33f9ab96efe42d").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when sending a invalid signature", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Hub-Signature-256", "sha256=⚠️").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})

	Context("slack", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{
				factory.SourceP(),
				factory.SourceP(func(o *entities.Source) { o.Config.HTTP.Path = "/validate-tolerance" }),
			},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "slack",
					ProviderConfig: map[string]interface{}{
						"secret": "test-slack-secret",
					},
				}),
			),
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[1].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "slack",
					ProviderConfig: map[string]interface{}{
						"secret":           "test-slack-secret",
						"tolerance_window": 300,
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should succeed", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Slack-Signature", "v0=159ef3656f49c5461cbbe81007f6fb29b74b1f169b9a07ee3b6b8e9cd2654865").
				SetHeader("X-Slack-Request-Timestamp", "1531420618").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail due to tolerance check", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Slack-Signature", "v0=159ef3656f49c5461cbbe81007f6fb29b74b1f169b9a07ee3b6b8e9cd2654865").
				Post("/validate-tolerance")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))

			resp, err = proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Slack-Signature", "v0=159ef3656f49c5461cbbe81007f6fb29b74b1f169b9a07ee3b6b8e9cd2654865").
				SetHeader("X-Slack-Request-Timestamp", "1531420618").
				Post("/validate-tolerance")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when missing signature header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when X-Slack-Signature doesn't match", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Slack-Signature", "v0=test").
				SetHeader("X-Slack-Request-Timestamp", "1531420618").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when X-Slack-Request-Timestamp doesn't match", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Slack-Signature", "v0=159ef3656f49c5461cbbe81007f6fb29b74b1f169b9a07ee3b6b8e9cd2654865").
				SetHeader("X-Slack-Request-Timestamp", "0").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})

	Context("stripe", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{
				factory.SourceP(),
				factory.SourceP(func(o *entities.Source) {
					o.Config.HTTP.Path = "/v2"
				}),
			},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "stripe",
					ProviderConfig: map[string]interface{}{
						"secret":           "test-stripe-secret",
						"tolerance_window": 300,
					},
				}),
			),
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[1].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "stripe",
					ProviderConfig: map[string]interface{}{
						"secret":           "test-stripe-secret",
						"tolerance_window": 0,
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should succeed", func() {
			now := time.Now()
			signature := utils.HmacEncode(
				"sha-256",
				[]byte("test-stripe-secret"),
				[]byte(fmt.Sprintf(`%d.{"event_type": "foo.bar","data": {"key": "value"}}`, now.Unix())),
				"hex")
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s,v0=cccc", now.Unix(), signature)).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should succeed when tolerance is set to 0", func() {
			now := time.Now().Add(-time.Hour)
			signature := utils.HmacEncode(
				"sha-256",
				[]byte("test-stripe-secret"),
				[]byte(fmt.Sprintf(`%d.{"event_type": "foo.bar","data": {"key": "value"}}`, now.Unix())),
				"hex")
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s,v0=cccc", now.Unix(), signature)).
				Post("/v2")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail when t is older than tolerance", func() {
			now := time.Now().Add(time.Second * -300)
			signature := utils.HmacEncode(
				"sha-256",
				[]byte("test-stripe-secret"),
				[]byte(fmt.Sprintf(`%d.{"event_type": "foo.bar","data": {"key": "value"}}`, now.Unix())),
				"hex")
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s,v0=cccc", now.Unix(), signature)).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when missing signature header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when signature `v1` doesn't match", func() {
			now := time.Now()
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("Stripe-Signature", fmt.Sprintf("t=%d,v1=test,v0=cccc", now.Unix())).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when signature `t` doesn't match", func() {
			now := time.Now()
			signature := utils.HmacEncode(
				"sha-256",
				[]byte("test-stripe-secret"),
				[]byte(fmt.Sprintf(`%d.{"event_type": "foo.bar","data": {"key": "value"}}`, now.Unix())),
				"hex")
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s,v0=cccc", now.Unix()+1, signature)).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})

	Context("gitlab", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "gitlab",
					ProviderConfig: map[string]interface{}{
						"secret": "test-gitlab-secret",
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should succeed", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Gitlab-Token", "test-gitlab-secret").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail when missing X-Gitlab-Token", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})

	Context("openapi", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{
				factory.SourceP(),
				factory.SourceP(func(o *entities.Source) { o.Config.HTTP.Path = "/validate-tolerance" }),
			},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "openai",
					ProviderConfig: map[string]interface{}{
						"secret":           "whsec_Nnl1eM1Ll9kclCuoQM14OMc6R1gI7hTPtoOoD/gtcJI=",
						"tolerance_window": 0,
					},
				}),
			),
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[1].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "openai",
					ProviderConfig: map[string]interface{}{
						"secret":           "whsec_Nnl1eM1Ll9kclCuoQM14OMc6R1gI7hTPtoOoD/gtcJI=",
						"tolerance_window": 300,
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		var sign = func(id, timestamp, payload, secret string) string {
			secret = strings.TrimPrefix(secret, "whsec_")
			secretBytes, err := base64.StdEncoding.DecodeString(secret)
			if err != nil {
				panic(err)
			}
			message := fmt.Sprintf("%s.%s.%s", id, timestamp, payload)
			return utils.HmacEncode("sha-256", secretBytes, []byte(message), "base64")
		}

		It("should succeed", func() {
			timestamp := fmt.Sprintf("%d", time.Now().Unix())
			signature := sign("wh_6943c6048e088190b1257ed5df29777e", timestamp, `{"event_type": "foo.bar","data": {"key": "value"}}`, "whsec_Nnl1eM1Ll9kclCuoQM14OMc6R1gI7hTPtoOoD/gtcJI=")
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("webhook-id", "wh_6943c6048e088190b1257ed5df29777e").
				SetHeader("webhook-timestamp", timestamp).
				SetHeader("webhook-signature", "v1,test v1,"+signature).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail due to tolerance check", func() {
			now := time.Now().Add(time.Second * -301)
			timestamp := fmt.Sprintf("%d", now.Unix())
			signature := sign("wh_6943c6048e088190b1257ed5df29777e", timestamp, `{"event_type": "foo.bar","data": {"key": "value"}}`, "whsec_Nnl1eM1Ll9kclCuoQM14OMc6R1gI7hTPtoOoD/gtcJI=")

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("webhook-id", "wh_6943c6048e088190b1257ed5df29777e").
				SetHeader("webhook-signature", "v1,test v1,"+signature).
				Post("/validate-tolerance")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))

			resp, err = proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("webhook-id", "wh_6943c6048e088190b1257ed5df29777e").
				SetHeader("webhook-timestamp", timestamp).
				SetHeader("webhook-signature", "v1,test v1,"+signature).
				Post("/validate-tolerance")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when missing headers", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})

	Context("okta", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{factory.SourceP(func(o *entities.Source) {
				o.Config.HTTP.Methods = []string{"GET", "POST"}
			})},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "okta",
					ProviderConfig: map[string]interface{}{
						"authentication_field":  "X-Okta-Token",
						"authentication_secret": "test-secret",
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		Context("verification challenge", func() {
			It("should succeed", func() {
				resp, err := proxyClient.R().
					SetHeader("x-okta-verification-challenge", "aGMfEUB5SlPbLiNOC1-aYFSAAlxdZ-x8GZbsbi1f").
					SetHeader("X-Okta-Token", "test-secret").
					Get("/")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"verification":"aGMfEUB5SlPbLiNOC1-aYFSAAlxdZ-x8GZbsbi1f"}`, string(resp.Body()))
			})
		})

		It("should succeed", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Okta-Token", "test-secret").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail when missing authentication header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})

	Context("zendesk", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{
				factory.SourceP(),
				factory.SourceP(func(o *entities.Source) { o.Config.HTTP.Path = "/validate-tolerance" }),
			},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "zendesk",
					ProviderConfig: map[string]interface{}{
						"secret": "dGhpc19zZWNyZXRfaXNfZm9yX3Rlc3Rpbmdfb25seQ==",
					},
				}),
			),
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[1].ID),
				factory.WithPluginName("integration-auth"),
				factory.WithPluginConfig(integration_auth.Config{
					Provider: "zendesk",
					ProviderConfig: map[string]interface{}{
						"secret":           "dGhpc19zZWNyZXRfaXNfZm9yX3Rlc3Rpbmdfb25seQ==",
						"tolerance_window": 300,
					},
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should succeed", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Zendesk-Webhook-Signature", "qtmu3iClLKujnujOBkwe7yjI34o5CzsLD04Uy4LbE2A=").
				SetHeader("X-Zendesk-Webhook-Signature-Timestamp", "1531420618").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should fail due to tolerance check", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Zendesk-Webhook-Signature", "qtmu3iClLKujnujOBkwe7yjI34o5CzsLD04Uy4LbE2A=").
				Post("/validate-tolerance")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))

			resp, err = proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Zendesk-Webhook-Signature", "qtmu3iClLKujnujOBkwe7yjI34o5CzsLD04Uy4LbE2A=").
				SetHeader("X-Zendesk-Webhook-Signature-Timestamp", "1531420618").
				Post("/validate-tolerance")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when missing signature header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when X-Slack-Signature doesn't match", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Zendesk-Webhook-Signature", "test").
				SetHeader("X-Zendesk-Webhook-Signature-Timestamp", "1531420618").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should fail when X-Slack-Request-Timestamp doesn't match", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("X-Zendesk-Webhook-Signature", "qtmu3iClLKujnujOBkwe7yjI34o5CzsLD04Uy4LbE2A=").
				SetHeader("X-Zendesk-Webhook-Signature-Timestamp", "0").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})
})

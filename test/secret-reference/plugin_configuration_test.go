package secret_reference

import (
	"encoding/json"
	"fmt"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/mock"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("Plugin Configuration", Ordered, func() {

	plugin.RegisterPlugin(plugin.TypeInbound, "mock", func() plugin.Plugin { return &mock.Plugin{} })

	Context("sanity", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.TestEntities{
			Sources: []*entities.Source{factory.Source()},
		}

		entitiesConfig.Plugins = []*entities.Plugin{
			factory.Plugin("mock", func(o *entities.Plugin) {
				o.Config = map[string]interface{}{
					"status": 200,
					"headers": map[string]interface{}{
						"Content-Type": "application/json",
						"x-k1":         "{secret://aws/webhookx/config.k1}",
						"x-k2":         "{secret://aws/webhookx/config.k2}",
						"x-k3":         "{secret://aws/webhookx/config.k3}",
					},
					"body": "hello world",
				}
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
		}

		BeforeAll(func() {
			data := map[string]interface{}{
				"k1": "v1",
				"k2": "v2",
				"k3": "v3",
			}
			b, err := json.Marshal(data)
			assert.NoError(GinkgoT(), err)
			smClient := helper.SecretManangerClient()
			err = upsertSecret(smClient, "webhookx/config", string(b))
			assert.NoError(GinkgoT(), err)

			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()
			app = helper.MustStart(map[string]string{
				"AWS_ACCESS_KEY_ID":          "test",
				"AWS_SECRET_ACCESS_KEY":      "test",
				"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
				"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",
			}, helper.WithLicenser(&helper.MockLicenser{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("plugin configuration's references should be resolved", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.Equal(GinkgoT(), "application/json", resp.Header().Get("Content-Type"))
			assert.Equal(GinkgoT(), "v1", resp.Header().Get("x-k1"))
			assert.Equal(GinkgoT(), "v2", resp.Header().Get("x-k2"))
			assert.Equal(GinkgoT(), "v3", resp.Header().Get("x-k3"))
			assert.Equal(GinkgoT(), "hello world", string(resp.Body()))
		})

	})

	Context("errors", func() {
		It("should fail for references that doesn't exist", func() {
			entitiesConfig := helper.TestEntities{}
			entitiesConfig.AddSource(factory.Source(func(o *entities.Source) {
				o.Plugins = []*entities.Plugin{
					factory.Plugin("mock", func(o *entities.Plugin) {
						o.Config = map[string]interface{}{
							"status": 200,
							"headers": map[string]interface{}{
								"Content-Type": "application/json",
								"x-notexist":   "{secret://aws/notexist}",
							},
							"body": "hello world",
						}
					}),
				}
			}))
			helper.InitDB(true, &entitiesConfig)

			_, err := helper.Start(map[string]string{
				"AWS_ACCESS_KEY_ID":          "test",
				"AWS_SECRET_ACCESS_KEY":      "test",
				"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
				"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",
			}, helper.WithLicenser(&helper.MockLicenser{}))
			assert.EqualError(GinkgoT(), err,
				fmt.Sprintf("failed to build plugin iterator: failed to load plugins: plugin{id=%s} configuration reference resolve failed: property \"headers.x-notexist\" resolve error: failed to resolve reference value '{secret://aws/notexist}': secret not found", entitiesConfig.Sources[0].Plugins[0].ID))
		})

		It("should fails for invalid reference", func() {
			entitiesConfig := helper.TestEntities{}
			entitiesConfig.AddSource(factory.Source(func(o *entities.Source) {
				o.Plugins = []*entities.Plugin{
					factory.Plugin("mock", func(o *entities.Plugin) {
						o.Config = map[string]interface{}{
							"status": 200,
							"headers": map[string]interface{}{
								"Content-Type": "application/json",
								"x-invalid":    "{secret://aws/}",
							},
							"body": "hello world",
						}
					}),
				}
			}))
			helper.InitDB(true, &entitiesConfig)

			_, err := helper.Start(map[string]string{
				"AWS_ACCESS_KEY_ID":          "test",
				"AWS_SECRET_ACCESS_KEY":      "test",
				"WEBHOOKX_SECRET_AWS_REGION": "us-east-1",
				"WEBHOOKX_SECRET_AWS_URL":    "http://localhost:4566",
			}, helper.WithLicenser(&helper.MockLicenser{}))
			assert.EqualError(GinkgoT(), err,
				fmt.Sprintf("failed to build plugin iterator: failed to load plugins: plugin{id=%s} configuration reference resolve failed: property \"headers.x-invalid\" parse error: invalid reference: \"invalid reference name\"", entitiesConfig.Sources[0].Plugins[0].ID))
		})
	})
})

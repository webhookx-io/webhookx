package plugins_test

import (
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	key_auth "github.com/webhookx-io/webhookx/plugins/key-auth"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("key-auth", Ordered, func() {
	Context("", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("key-auth"),
				factory.WithPluginConfig(key_auth.Config{
					ParamName:      "apikey",
					ParamLocations: []string{"query", "header"},
					Key:            "thisisasecret",
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

		It("should pass when passing right auth in header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("apikey", "thisisasecret").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should pass when passing right auth in query", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("apikey", "thisisasecret").
				Post("/?apikey=thisisasecret")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should deny when missing auth", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should deny when passing wrong auth in header", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("apikey", "wrongkey").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should deny when passing wrong auth in query", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/?apikey=wrongkey")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})
})

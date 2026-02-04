package plugins_test

import (
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	basic_auth "github.com/webhookx-io/webhookx/plugins/basic-auth"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("basic-auth", Ordered, func() {
	Context("", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources: []*entities.Source{
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/"
					o.Plugins = append(o.Plugins, factory.Plugin("basic-auth",
						factory.WithPluginConfig(basic_auth.Config{
							Username: "username",
							Password: "password",
						}),
					))
				}),
				factory.Source(func(o *entities.Source) {
					o.Config.HTTP.Path = "/empty-password"
					o.Plugins = append(o.Plugins, factory.Plugin("basic-auth",
						factory.WithPluginConfig(basic_auth.Config{
							Username: "username",
							Password: "",
						}),
					))
				})},
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(nil))
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should pass when passing right auth", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetBasicAuth("username", "password").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("should pass when passing empty password", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetBasicAuth("username", "").
				Post("/empty-password")
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

		It("should deny when passing wrong auth", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetBasicAuth("username", "").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})

		It("should deny when passing inavlid auth", func() {
			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				SetHeader("Authorization", "Basic <credentials>").
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 401, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"Unauthorized"}`, string(resp.Body()))
		})
	})
})

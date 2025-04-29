package plugins

import (
	"context"
	"github.com/webhookx-io/webhookx/plugins/wasm"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("wasm", Ordered, func() {

	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginEndpointID(entitiesConfig.Endpoints[0].ID),
				factory.WithPluginName("wasm"),
				factory.WithPluginConfig(wasm.Config{
					File: test.FilePath("plugins/testdata/index.wasm"),
				}),
			),
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
				"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
				"WEBHOOKX_WORKER_ENABLED": "true",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
					    "event_type": "foo.bar",
					    "data": {"key": "value"}
					}`).
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Attempts.List(context.TODO(), &query.AttemptQuery{})
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusSuccess
			}, time.Second*5, time.Second)

			assert.Equal(GinkgoT(), entitiesConfig.Endpoints[0].ID, attempt.EndpointId)

			assert.Equal(GinkgoT(), "PUT", attempt.Request.Method)
			assert.Equal(GinkgoT(), "http://localhost:9999/anything?debug=true", attempt.Request.URL)

			var attemptDetail *entities.AttemptDetail
			assert.Eventually(GinkgoT(), func() bool {
				val, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
				if err != nil || val == nil {
					return false
				}
				attemptDetail = val
				return true
			}, time.Second*5, time.Second)

			assert.Equal(GinkgoT(), "bar", attemptDetail.RequestHeaders["Foo"])
			assert.Equal(GinkgoT(), `{"key": "value", "other": "other-value"}`, *attemptDetail.RequestBody)
		})
	})
})

package plugins

import (
	"context"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/plugins/webhookx_signature"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"strconv"
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

var _ = Describe("webhookx-signature", Ordered, func() {

	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		endpoint := factory.EndpointP()
		endpoint.Request.Headers = map[string]string{
			"foo": "bar",
		}

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{endpoint},
			Sources:   []*entities.Source{factory.SourceP()},
		}

		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginEndpointID(entitiesConfig.Endpoints[0].ID),
				factory.WithPluginName("webhookx-signature"),
				factory.WithPluginConfig(webhookx_signature.Config{
					SigningSecret: "abcdefg",
				}),
			),
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			now := time.Now()
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
					    "event_type": "foo.bar",
					    "data": {"key": "value"}
					}`).
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			var event *entities.Event
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Events.List(context.TODO(), &query.EventQuery{})
				if err != nil || len(list) != 1 {
					return false
				}
				event = list[0]
				return true
			}, time.Second*5, time.Second)
			assert.True(GinkgoT(), event.IngestedAt.UnixMilli() >= now.UnixMilli())

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

			// attempt.request
			assert.Equal(GinkgoT(), "POST", attempt.Request.Method)
			assert.Equal(GinkgoT(), "http://localhost:9999/anything", attempt.Request.URL)
			assert.Nil(GinkgoT(), attempt.Request.Headers)
			assert.Nil(GinkgoT(), attempt.Request.Body)

			// attempt.resposne
			assert.True(GinkgoT(), attempt.Response.Latency > 0)
			assert.Equal(GinkgoT(), 200, attempt.Response.Status)
			assert.Nil(GinkgoT(), attempt.Response.Headers)
			assert.Nil(GinkgoT(), attempt.Response.Body)

			var attemptDetail *entities.AttemptDetail
			assert.Eventually(GinkgoT(), func() bool {
				val, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
				if err != nil || val == nil {
					return false
				}
				attemptDetail = val
				return true
			}, time.Second*5, time.Second)

			// attemptDetail.request
			assert.Equal(GinkgoT(), "application/json; charset=utf-8", attemptDetail.RequestHeaders["Content-Type"])
			assert.Equal(GinkgoT(), "WebhookX/"+config.VERSION, attemptDetail.RequestHeaders["User-Agent"])
			assert.Regexp(GinkgoT(), "v1=[0-9a-f]{64}", attemptDetail.RequestHeaders["Webhookx-Signature"])
			timestamp := attemptDetail.RequestHeaders["Webhookx-Timestamp"]
			assert.True(GinkgoT(), utils.Must(strconv.ParseInt(timestamp, 10, 0)) >= attempt.AttemptedAt.Unix())
			assert.Equal(GinkgoT(), `{"key": "value"}`, *attemptDetail.RequestBody)
			assert.Equal(GinkgoT(), "bar", attemptDetail.RequestHeaders["Foo"])
		})
	})
})

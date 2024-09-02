package delivery

import (
	"context"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
	"time"
)

var _ = Describe("delivery", Ordered, func() {

	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
			Sources:   []*entities.Source{helper.DefaultSource()},
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
					    "data": {
							"key": "value"
						}
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
			}, time.Second*15, time.Second)

			assert.Equal(GinkgoT(), entitiesConfig.Endpoints[0].ID, attempt.EndpointId)
			assert.Equal(GinkgoT(), &entities.AttemptRequest{
				Method: "POST",
				URL:    "http://localhost:9999/anything",
				Headers: map[string]string{
					"Content-Type": "application/json; charset=utf-8",
					"User-Agent":   "WebhookX/" + config.VERSION,
				},
				Body: utils.Pointer(`{"key": "value"}`),
			}, attempt.Request)
			assert.Equal(GinkgoT(), 200, attempt.Response.Status)
		})
	})

	Context("retries (timeout)", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB
		var endpoint = helper.DefaultEndpoint()

		BeforeAll(func() {
			endpoint.Request.Timeout = 1
			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{endpoint},
				Sources:   []*entities.Source{helper.DefaultSource()},
			}
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

		It("all tries are exhausted", func() {
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
				    "event_type": "foo.bar",
				    "data": {
						"key": "value"
					}
				}`).
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			time.Sleep(time.Second * 10)

			attempts, err := db.Attempts.List(context.TODO(), &query.AttemptQuery{})
			assert.NoError(GinkgoT(), err)
			assert.EqualValues(GinkgoT(), 3, len(attempts))
			for i, e := range attempts {
				assert.Equal(GinkgoT(), "TIMEOUT", *e.ErrorCode)
				assert.Equal(GinkgoT(), "FAILED", e.Status)
				assert.Equal(GinkgoT(), i+1, e.AttemptNumber)
				assert.Nil(GinkgoT(), e.Response)
			}
		})
	})

	Context("retries (endpoint disabled)", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB
		var endpoint = helper.DefaultEndpoint()

		BeforeAll(func() {
			endpoint.Retry.Config.Attempts = []int64{3, 1, 1}
			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{endpoint},
				Sources:   []*entities.Source{helper.DefaultSource()},
			}
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

		It("stop retry when endpoint is disabled", func() {
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
					    "event_type": "foo.bar",
					    "data": {
							"key": "value"
						}
					}`).
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			// disable endpoint
			entity, err := db.Endpoints.Get(context.TODO(), endpoint.ID)
			assert.NoError(GinkgoT(), err)
			entity.Enabled = false
			assert.NoError(GinkgoT(), db.Endpoints.Update(context.TODO(), entity))

			time.Sleep(time.Second * 10)

			attempts, err := db.Attempts.List(context.TODO(), &query.AttemptQuery{})
			assert.NoError(GinkgoT(), err)
			assert.EqualValues(GinkgoT(), 1, len(attempts))
			attempt := attempts[0]
			assert.Equal(GinkgoT(), "ENDPOINT_DISABLED", *attempt.ErrorCode)
			assert.Equal(GinkgoT(), "CANCELED", attempt.Status)
			assert.Equal(GinkgoT(), 1, attempt.AttemptNumber)
			assert.Nil(GinkgoT(), attempt.Response)
		})
	})
})

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delivery Suite")
}

package delivery

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/worker/circuitbreaker"
)

var _ = Describe("CircuitBreaker", Ordered, func() {
	Context("", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB
		var flushInterval time.Duration
		var endpoint = factory.Endpoint(func(o *entities.Endpoint) {
			o.Request.URL = "http://localhost:9999/status/500"
			o.Retry.Config.Attempts = []int64{0}
		})

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{endpoint},
			Sources:   []*entities.Source{factory.Source()},
		}

		BeforeAll(func() {
			circuitbreaker.DefaultFlushInterval = time.Second
			flushInterval = circuitbreaker.DefaultFlushInterval

			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = helper.MustStart(map[string]string{
				"WEBHOOKX_WORKER_CIRCUITBREAKER_ENABLED":                   "true",
				"WEBHOOKX_WORKER_CIRCUITBREAKER_WINDOW_SIZE":               "3600",
				"WEBHOOKX_WORKER_CIRCUITBREAKER_FAILURE_RATE_THRESHOLD":    "90",
				"WEBHOOKX_WORKER_CIRCUITBREAKER_MINIMUM_REQUEST_THRESHOLD": "10",
			})

			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
			circuitbreaker.DefaultFlushInterval = flushInterval
		})

		It("endpoint should be disabled", func() {
			for i := 0; i < 10; i++ {
				resp, err := proxyClient.R().
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
			}
			assert.Eventually(GinkgoT(), func() bool {
				q := dao.AttemptQuery{
					Status: new(entities.AttemptStatusFailure),
				}
				n, err := db.Attempts.Count(context.TODO(), q.ToQuery())
				assert.NoError(GinkgoT(), err)
				return n == 10
			}, time.Second*3, time.Millisecond*100)

			time.Sleep(time.Second * 1)
			app.Scheduler().RunNow("worker.detectEndpointHealthy")

			endpoint, err := db.Endpoints.Get(context.TODO(), endpoint.ID)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), false, endpoint.Enabled)
		})

	})
})

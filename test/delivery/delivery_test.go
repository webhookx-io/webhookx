package delivery

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/test/helper/factory"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("delivery", Ordered, func() {
	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources:   []*entities.Source{factory.Source()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{{
			ID:         utils.KSUID(),
			EndpointId: utils.Pointer(entitiesConfig.Endpoints[0].ID),
			Name:       "webhookx-signature",
			Enabled:    true,
			Config:     map[string]interface{}{"key": "abcdefg"},
		}}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
			now := time.Now()

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)
			event, err := db.Events.Get(context.TODO(), eventId)
			assert.NoError(GinkgoT(), err)
			assert.NotNil(GinkgoT(), event)
			assert.True(GinkgoT(), event.IngestedAt.UnixMilli() >= now.UnixMilli())

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				q := query.AttemptQuery{}
				q.EventId = &eventId
				list, err := db.Attempts.List(context.TODO(), &q)
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

			time.Sleep(time.Second)
			attemptDetail, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
			assert.NoError(GinkgoT(), err)
			assert.NotNil(GinkgoT(), attemptDetail)

			// attemptDetail.request
			assert.Equal(GinkgoT(), "application/json; charset=utf-8", attemptDetail.RequestHeaders["Content-Type"])
			assert.Equal(GinkgoT(), "WebhookX/"+webhookx.VERSION, attemptDetail.RequestHeaders["User-Agent"])
			assert.Regexp(GinkgoT(), "v1=[0-9a-f]{64}", attemptDetail.RequestHeaders["Webhookx-Signature"])
			timestamp := attemptDetail.RequestHeaders["Webhookx-Timestamp"]
			assert.True(GinkgoT(), utils.Must(strconv.ParseInt(timestamp, 10, 0)) >= attempt.AttemptedAt.Unix())
			assert.Equal(GinkgoT(), attempt.EventId, attemptDetail.RequestHeaders["Webhookx-Event-Id"])
			assert.Equal(GinkgoT(), attempt.ID, attemptDetail.RequestHeaders["Webhookx-Delivery-Id"])
			assert.Equal(GinkgoT(), `{"key": "value"}`, *attemptDetail.RequestBody)
		})
	})

	Context("retries (timeout)", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		BeforeAll(func() {
			entitiesConfig := helper.TestEntities{
				Endpoints: []*entities.Endpoint{factory.Endpoint(func(o *entities.Endpoint) {
					o.Request.Timeout = 1
					o.Request.URL = "http://localhost:9999/delay/1"
					o.Retry.Config.Attempts = []int64{0, 0, 0}
				})},
				Sources: []*entities.Source{factory.Source()},
			}
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("all tries are exhausted", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

			time.Sleep(time.Second * 4)

			q := query.AttemptQuery{}
			q.EventId = &eventId
			attempts, err := db.Attempts.List(context.TODO(), &q)
			assert.NoError(GinkgoT(), err)
			assert.EqualValues(GinkgoT(), 3, len(attempts))
			for i, e := range attempts {
				assert.NotNil(GinkgoT(), e.ErrorCode, "ErrorCode is not nil: %v", e.ErrorCode)
				assert.Equal(GinkgoT(), "TIMEOUT", *e.ErrorCode)
				assert.Equal(GinkgoT(), "FAILED", e.Status)
				assert.Equal(GinkgoT(), i+1, e.AttemptNumber)

				assert.Equal(GinkgoT(), i+1 == len(attempts), e.Exhausted) // exhausted should be true when it's the last attempt
				if i == 0 {
					assert.Equal(GinkgoT(), entities.AttemptTriggerModeInitial, e.TriggerMode)
				} else {
					assert.Equal(GinkgoT(), entities.AttemptTriggerModeAutomatic, e.TriggerMode)
				}
				assert.Nil(GinkgoT(), e.Response)

				attemptDetail, err := db.AttemptDetails.Get(context.TODO(), e.ID)
				assert.NoError(GinkgoT(), err)
				assert.Nil(GinkgoT(), attemptDetail.ResponseHeaders)
				assert.Nil(GinkgoT(), attemptDetail.ResponseBody)
			}
		})
	})

	Context("retries (endpoint disabled)", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB
		var endpoint = factory.Endpoint()

		BeforeAll(func() {
			endpoint.Retry.Config.Attempts = []int64{3, 1, 1}
			entitiesConfig := helper.TestEntities{
				Endpoints: []*entities.Endpoint{endpoint},
				Sources:   []*entities.Source{factory.Source()},
			}
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
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
			assert.Nil(GinkgoT(), attempt.Request)
			assert.Nil(GinkgoT(), attempt.Response)

			attemptDetail, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
			assert.NoError(GinkgoT(), err)
			// Disable endpoint will not create request
			assert.Nil(GinkgoT(), attemptDetail)
		})
	})

	Context("schedule task", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		BeforeAll(func() {
			entitiesConfig := helper.TestEntities{
				Endpoints: []*entities.Endpoint{factory.Endpoint(func(o *entities.Endpoint) {
					o.Retry.Config.Attempts = []int64{
						int64(constants.TaskQueuePreScheduleTimeWindow.Seconds()) + 3}
				})},
				Sources: []*entities.Source{factory.Source()},
			}
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("schedule task when conditions met", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

			query := query.AttemptQuery{}
			query.EventId = &eventId
			list, err := db.Attempts.List(context.TODO(), &query)
			assert.NoError(GinkgoT(), err)
			assert.EqualValues(GinkgoT(), 1, len(list))
			assert.Equal(GinkgoT(), entities.AttemptStatusInit, list[0].Status) // should not be enqueued

			result, err := db.DB.Exec("UPDATE attempts set scheduled_at = $1, created_at = created_at - INTERVAL '30 SECOND' where id = $2", time.Now(), list[0].ID)
			assert.NoError(GinkgoT(), err)
			row, err := result.RowsAffected()
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), int64(1), row)

			task := app.Scheduler().GetTask("worker.requeue")
			assert.NotNil(GinkgoT(), task)
			task.Do() // load db data that meets the conditions into task queue

			assert.Eventually(GinkgoT(), func() bool {
				model, err := db.Attempts.Get(context.TODO(), list[0].ID)
				assert.NoError(GinkgoT(), err)
				return model.Status == entities.AttemptStatusSuccess
			}, time.Second*3, time.Millisecond*100)

		})
	})

	Context("rate limit", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB
		period := 5

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint(func(o *entities.Endpoint) {
				o.RateLimit = &entities.RateLimit{
					Quota:  3,
					Period: period,
				}
			})},
			Sources: []*entities.Source{factory.Source()},
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("rate limiting", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)

			for i := 1; i <= 4; i++ {
				resp, err := proxyClient.R().
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
			}

			assert.Eventually(GinkgoT(), func() bool {
				matched, err := helper.FileHasLine(helper.LogFile, "^.*rate limit.*$")
				return err == nil && matched
			}, time.Second*5, time.Second)

			// wait for attempt to be retried after rate limiting is reset
			time.Sleep(time.Second * time.Duration(period) * 2)

			q := query.AttemptQuery{}
			q.EndpointId = &entitiesConfig.Endpoints[0].ID
			q.Status = utils.Pointer(entities.AttemptStatusSuccess)
			count, err := db.Attempts.Count(context.TODO(), q.WhereMap())
			assert.NoError(GinkgoT(), err)
			assert.EqualValues(GinkgoT(), 4, count)

		})
	})

	Context("unique_id", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources:   []*entities.Source{factory.Source()},
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should de-duplicate events by unique_id", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
			for i := 1; i <= 2; i++ {
				resp, err := proxyClient.R().
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}, "unique_id":"key1"}`).
					Post("/")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
			}
			n, err := db.Events.Count(context.TODO(), nil)
			assert.NoError(GinkgoT(), err)
			assert.EqualValues(GinkgoT(), 1, n)
		})
	})
})

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Delivery Suite")
}

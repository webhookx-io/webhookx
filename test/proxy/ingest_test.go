package proxy

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("ingest", Ordered, func() {

	Context("sanity", func() {

		var proxyClient *resty.Client
		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources: []*entities.Source{
				factory.Source(),
				factory.Source(
					func(o *entities.Source) {
						o.Config.HTTP.Path = "/custom-response"
						o.Config.HTTP.Response = &entities.CustomResponse{
							Code:        201,
							ContentType: "application/xml",
							Body:        "<message>ok</message>",
						}
					}),
			},
		}
		entitiesConfig.Sources[0].Async = true

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_WORKER_ENABLED": "false",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			resp, err := proxyClient.R().
				SetBody(`{
					    "event_type": "foo.bar",
					    "data": {
							"key": "value"
						}
					}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.NotEmpty(GinkgoT(), resp.Header().Get("X-Webhookx-Event-Id"))

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Attempts.List(context.TODO(), &query.AttemptQuery{})
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusQueued
			}, time.Second*15, time.Second)
		})

		It("custom response", func() {
			resp, err := proxyClient.R().
				SetBody(`{
					    "event_type": "foo.bar",
					    "data": {
							"key": "value"
						}
					}`).
				Post("/custom-response")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 201, resp.StatusCode())
			assert.Equal(GinkgoT(), "application/xml", resp.Header().Get("Content-Type"))
			assert.Equal(GinkgoT(), "<message>ok</message>", string(resp.Body()))
			assert.NotEmpty(GinkgoT(), resp.Header().Get("X-Webhookx-Event-Id"))
		})

	})

	Context("queue disabled", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources:   []*entities.Source{factory.Source()},
		}
		entitiesConfig.Sources[0].Async = true

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_PROXY_QUEUE_TYPE": "off",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("returns HTTP 500", func() {
			helper.TruncateFile(helper.LogFile)
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
					    "event_type": "foo.bar",
					    "data": {
							"key": "value"
						}
					}`).
					Post("/")
				return err == nil && resp.StatusCode() == 500
			}, time.Second*5, time.Second)
			matched, err := helper.FileHasLine(helper.LogFile, "^.*failed to ingest event: queue is disabled$")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})

	})
})

package proxy

import (
	"context"
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
	"time"
)

var _ = Describe("ingest", Ordered, func() {

	Context("sanity", func() {

		var proxyClient *resty.Client
		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Sources[0].Async = true

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_PROXY_LISTEN": "0.0.0.0:8081",
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
				return attempt.Status == entities.AttemptStatusQueued
			}, time.Second*15, time.Second)
		})
	})

	Context("queue disabled", func() {
		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Sources[0].Async = true

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_PROXY_LISTEN":     "0.0.0.0:8081",
				"WEBHOOKX_PROXY_QUEUE_TYPE": "off",
				"WEBHOOKX_LOG_FILE":         "webhookx.log",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("returns HTTP 500", func() {
			helper.TruncateFile("webhookx.log")
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
			matched, err := helper.FileHasLine("webhookx.log", "^.*failed to ingest event: queue is disabled$")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})

	})
})

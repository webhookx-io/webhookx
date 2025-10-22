package delivery

import (
	"context"
	"crypto/tls"
	"github.com/elazarl/goproxy"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

type NoopLogger struct{}

func (NoopLogger) Printf(format string, v ...any) {}

var _ = Describe("http proxy", Ordered, func() {
	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		var proxyServer *http.Server
		var reverseServer *http.Server

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Events = []string{"http"}
				}),
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Request.URL = "https://localhost:9443/anything"
					o.Events = []string{"https"}
				}),
			},
			Sources: []*entities.Source{factory.SourceP()},
		}

		BeforeAll(func() {
			deliverer.DefaultTLSConfig = &tls.Config{InsecureSkipVerify: true} // mock tls config
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			proxy := goproxy.NewProxyHttpServer()
			proxy.Logger = &NoopLogger{}
			proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm) // mitm
			proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
				resp.Header.Set("X-Proxied", "true")
				return resp
			})
			proxyServer = &http.Server{
				Addr:    ":20000",
				Handler: proxy,
			}
			go func() {
				log.Fatal(proxyServer.ListenAndServe())
			}()

			target, _ := url.Parse("http://localhost:9999")
			reverse := httputil.NewSingleHostReverseProxy(target)
			reverseServer = &http.Server{
				Addr:    ":9443",
				Handler: reverse,
			}
			go func() {
				log.Fatal(reverseServer.ListenAndServeTLS(test.FilePath("server.crt"), test.FilePath("server.key")))
			}()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_PROXY_LISTEN":           "0.0.0.0:8081",
				"WEBHOOKX_WORKER_ENABLED":         "true",
				"WEBHOOKX_WORKER_DELIVERER_PROXY": "http://localhost:20000",
			}))

		})

		AfterAll(func() {
			app.Stop()
			deliverer.DefaultTLSConfig = nil // reset  tls config
		})

		It("http delivery request should be proxied", Pending, func() {
			err := waitForServer("0.0.0.0:8081", time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "http","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

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

			detail, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
			assert.NoError(GinkgoT(), err)
			assert.NotNil(GinkgoT(), detail.RequestHeaders)
			assert.NotNil(GinkgoT(), detail.RequestBody)
			assert.NotNil(GinkgoT(), detail.ResponseHeaders)
			assert.NotNil(GinkgoT(), detail.ResponseBody)
			assert.Equal(GinkgoT(), "true", (*detail.ResponseHeaders)["X-Proxied"])
		})

		It("https delivery request should be proxied", func() {
			err := waitForServer("0.0.0.0:8081", time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "https","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

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

			assert.Equal(GinkgoT(), entitiesConfig.Endpoints[1].ID, attempt.EndpointId)

			// attempt.request
			assert.Equal(GinkgoT(), "POST", attempt.Request.Method)
			assert.Equal(GinkgoT(), "https://localhost:9443/anything", attempt.Request.URL)
			assert.Nil(GinkgoT(), attempt.Request.Headers)
			assert.Nil(GinkgoT(), attempt.Request.Body)

			// attempt.resposne
			assert.True(GinkgoT(), attempt.Response.Latency > 0)
			assert.Equal(GinkgoT(), 200, attempt.Response.Status)
			assert.Nil(GinkgoT(), attempt.Response.Headers)
			assert.Nil(GinkgoT(), attempt.Response.Body)

			detail, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
			assert.NoError(GinkgoT(), err)
			assert.NotNil(GinkgoT(), detail.RequestHeaders)
			assert.NotNil(GinkgoT(), detail.RequestBody)
			assert.NotNil(GinkgoT(), detail.ResponseHeaders)
			assert.NotNil(GinkgoT(), detail.ResponseBody)
			assert.Equal(GinkgoT(), "true", (*detail.ResponseHeaders)["X-Proxied"])
		})

	})
})

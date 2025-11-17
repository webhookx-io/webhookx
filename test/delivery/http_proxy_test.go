package delivery

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

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
)

type NoopLogger struct{}

func (NoopLogger) Printf(format string, v ...any) {}

func NewHttpProxyServer(addr string) *http.Server {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Logger = &NoopLogger{}
	proxy.OnRequest(goproxy.ReqHostIs("deny.localhost:443")).HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		return &goproxy.ConnectAction{
			Action: goproxy.ConnectHijack,
			Hijack: func(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
				body := `{"error": "Proxy Authentication Required", "code": 407}`
				resp := fmt.Sprintf("HTTP/1.1 407 Proxy Authentication Required\r\n"+
					"Content-Type: application/json\r\n"+
					"Content-Length: %d\r\n"+
					"\r\n%s", len(body), body)
				client.Write([]byte(resp))
			},
		}, host
	})
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		resp.Header.Set("X-Proxied", "true")
		return resp
	})
	return &http.Server{
		Addr:    addr,
		Handler: proxy,
	}
}

var _ = Describe("Proxy", Ordered, func() {

	var httpsBinURL = "https://localhost:9443"

	var httpProxyListen = "127.0.0.1:9901"
	var httpsProxyListen = "127.0.0.1:9902"
	var mtlsProxyListen = "localhost:9903"
	var httpProxyURL = "http://" + httpProxyListen
	var httpsProxyURL = "https://" + httpsProxyListen
	var mtlsProxyURL = "https://" + mtlsProxyListen

	BeforeAll(func() {
		deliverer.DefaultTLSConfig = &tls.Config{InsecureSkipVerify: true} // mock tls config
		httpProxyServer := NewHttpProxyServer(httpProxyListen)
		go func() {
			log.Fatal(httpProxyServer.ListenAndServe())
		}()
		httpsProxyServer := NewHttpProxyServer(httpsProxyListen)
		go func() {
			log.Fatal(httpsProxyServer.ListenAndServeTLS(test.FilePath("server.crt"), test.FilePath("server.key")))
		}()
		mTLScert, err := tls.LoadX509KeyPair(
			test.FilePath("fixtures/mtls/server.crt"),
			test.FilePath("fixtures/mtls/server.key"))
		if err != nil {
			panic(err)
		}
		pem, err := os.ReadFile(test.FilePath("fixtures/mtls/client-ca.crt"))
		if err != nil {
			panic(err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			panic("failed to append CA certs")
		}
		mtlsProxyServer := NewHttpProxyServer(mtlsProxyListen)
		mtlsProxyServer.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{mTLScert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
		}
		go func() {
			log.Fatal(mtlsProxyServer.ListenAndServeTLS("", ""))
		}()
	})

	AfterAll(func() {
		deliverer.DefaultTLSConfig = nil // reset  tls config
	})

	Context("HTTP URL", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Events = []string{"http"}
				}),
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Request.URL = httpsBinURL + "/anything"
					o.Events = []string{"https"}
				}),
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Request.URL = "https://deny.localhost"
					o.Events = []string{"deny"}
				}),
			},
			Sources: []*entities.Source{factory.SourceP()},
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_WORKER_DELIVERER_PROXY": httpProxyURL,
			}))

		})

		AfterAll(func() {
			app.Stop()
		})

		It("http delivery request should be proxied", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
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
			assert.NotNil(GinkgoT(), detail)
			assert.NotNil(GinkgoT(), detail.RequestHeaders)
			assert.NotNil(GinkgoT(), detail.RequestBody)
			assert.NotNil(GinkgoT(), detail.ResponseHeaders)
			assert.NotNil(GinkgoT(), detail.ResponseBody)
			assert.Equal(GinkgoT(), "true", (*detail.ResponseHeaders)["X-Proxied"])
		})

		It("https delivery request should be proxied", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
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

		It("should be failed when connect ", func() {
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "deny","data": {"key": "value"}}`).
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
				return attempt.Status == entities.AttemptStatusFailure
			}, time.Second*5, time.Second)

			assert.Equal(GinkgoT(), entitiesConfig.Endpoints[2].ID, attempt.EndpointId)

			// attempt.request
			assert.Equal(GinkgoT(), "POST", attempt.Request.Method)
			assert.Equal(GinkgoT(), "https://deny.localhost", attempt.Request.URL)
		})
	})

	Context("HTTPS URL", func() {
		Context("scenario: tls verify = false", func() {
			var proxyClient *resty.Client

			var app *app.Application
			var db *db.DB

			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{
					factory.EndpointP(func(o *entities.Endpoint) {
						o.Events = []string{"http"}
					}),
					factory.EndpointP(func(o *entities.Endpoint) {
						o.Request.URL = httpsBinURL + "/anything"
						o.Events = []string{"https"}
					}),
				},
				Sources: []*entities.Source{factory.SourceP()},
			}

			BeforeAll(func() {
				deliverer.DefaultTLSConfig = &tls.Config{InsecureSkipVerify: true} // mock tls config
				db = helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()

				app = utils.Must(helper.Start(map[string]string{
					"WEBHOOKX_WORKER_DELIVERER_PROXY":            httpsProxyURL,
					"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_VERIFY": "TRUE",
				}))

			})

			AfterAll(func() {
				app.Stop()
				deliverer.DefaultTLSConfig = nil // reset  tls config
			})

			It("http delivery request should be proxied", func() {
				err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
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
				err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
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

		Context("scenario: mTLS", func() {
			var proxyClient *resty.Client

			var app *app.Application
			var db *db.DB

			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{
					factory.EndpointP(func(o *entities.Endpoint) {
						o.Events = []string{"http"}
					}),
					factory.EndpointP(func(o *entities.Endpoint) {
						o.Request.URL = httpsBinURL + "/anything"
						o.Events = []string{"https"}
					}),
				},
				Sources: []*entities.Source{factory.SourceP()},
			}

			BeforeAll(func() {
				deliverer.DefaultTLSConfig = &tls.Config{InsecureSkipVerify: true} // mock tls config
				db = helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()

				app = utils.Must(helper.Start(map[string]string{
					"WEBHOOKX_WORKER_DELIVERER_PROXY":             mtlsProxyURL,
					"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CERT":    test.FilePath("fixtures/mtls/client.crt"),
					"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_KEY":     test.FilePath("fixtures/mtls/client.key"),
					"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CA_CERT": test.FilePath("fixtures/mtls/server-ca.crt"),
				}))

			})

			AfterAll(func() {
				app.Stop()
				deliverer.DefaultTLSConfig = nil // reset  tls config
			})

			It("http delivery request should be proxied", func() {
				err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
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

				time.Sleep(time.Millisecond * 100)
				detail, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
				assert.NoError(GinkgoT(), err)
				assert.NotNil(GinkgoT(), detail.RequestHeaders)
				assert.NotNil(GinkgoT(), detail.RequestBody)
				assert.NotNil(GinkgoT(), detail.ResponseHeaders)
				assert.NotNil(GinkgoT(), detail.ResponseBody)
				assert.Equal(GinkgoT(), "true", (*detail.ResponseHeaders)["X-Proxied"])
			})

			It("https delivery request should be proxied", func() {
				err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
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

	Context("error", func() {
		It("returns error when certificate not found", func() {
			_, err := helper.Start(map[string]string{
				"WEBHOOKX_WORKER_DELIVERER_PROXY":             mtlsProxyURL,
				"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CERT":    test.FilePath("fixtures/mtls/notfound.crt"),
				"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_KEY":     test.FilePath("fixtures/mtls/client.key"),
				"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CA_CERT": test.FilePath("fixtures/mtls/server-ca.crt"),
			})
			assert.Equal(GinkgoT(),
				fmt.Sprintf("failed to load client certificate: open %s: no such file or directory", test.FilePath("fixtures/mtls/notfound.crt")),
				err.Error())
		})
		It("returns error when ca cert not found", func() {
			_, err := helper.Start(map[string]string{
				"WEBHOOKX_WORKER_DELIVERER_PROXY":                 mtlsProxyURL,
				"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CLIENT_CERT": test.FilePath("fixtures/mtls/client.crt"),
				"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CLIENT_KEY":  test.FilePath("fixtures/mtls/client.key"),
				"WEBHOOKX_WORKER_DELIVERER_PROXY_TLS_CA_CERT":     test.FilePath("fixtures/mtls/notfound.crt"),
			})
			assert.Equal(GinkgoT(),
				fmt.Sprintf("failed to read ca certificate: open %s: no such file or directory", test.FilePath("fixtures/mtls/notfound.crt")),
				err.Error())
		})
	})
})

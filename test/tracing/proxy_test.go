package tracing

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"reflect"
	"time"
)

var _ = Describe("tracing proxy", Ordered, func() {
	endpoints := map[string]string{
		"grpc":          "localhost:4317",
		"http/protobuf": "http://localhost:4318/v1/traces",
	}
	for _, protocol := range []string{"grpc", "http/protobuf"} {
		Context(protocol, func() {
			var app *app.Application
			var proxyClient *resty.Client

			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
				Sources:   []*entities.Source{helper.DefaultSource()},
			}

			BeforeAll(func() {
				helper.InitOtelOutput()
				helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()

				envs := map[string]string{
					"WEBHOOKX_PROXY_LISTEN":                   "0.0.0.0:8081",
					"WEBHOOKX_TRACING_ENABLED":                "true",
					"WEBHOOKX_TRACING_SAMPLING_RATE":          "1.0",
					"WEBHOOKX_TRACING_ATTRIBUTES":             `{"env":"test"}`,
					"WEBHOOKX_TRACING_OPENTELEMETRY_PROTOCOL": protocol,
					"WEBHOOKX_TRACING_OPENTELEMETRY_ENDPOINT": endpoints[protocol],
				}
				app = utils.Must(helper.Start(envs))

			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity "+protocol, func() {
				var traceID = helper.GenerateTraceID()
				n, err := helper.FileCountLine(helper.OtelCollectorTracesFile)
				assert.Nil(GinkgoT(), err)
				n++
				fmt.Println("start line " + fmt.Sprint(n))

				expectedScopeNames := []string{
					"github.com/webhookx-io/webhookx",
					"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
				}

				entrypoint := map[string]string{
					"http.method":                  "POST",
					"http.scheme":                  "http",
					"http.target":                  "/",
					"http.status_code":             "200",
					"http.request_content_length":  "*",
					"http.response_content_length": "*",
					"user_agent.original":          "*",
					"net.host.name":                "localhost",
					"net.host.port":                "8081",
					"net.protocol.version":         "*",
					"net.sock.peer.addr":           "*",
					"net.sock.peer.port":           "*",
				}
				router := map[string]string{
					"source.id":           "*",
					"source.name":         "*",
					"source.workspace_id": "*",
					"source.async":        "false",
					"http.route":          "/",
				}
				expectedScopeSpans := map[string]map[string]string{
					"api.proxy":                 entrypoint,
					"proxy.handle":              router,
					"dispatcher.dispatch":       {},
					"dao.endpoints.list":        {},
					"db.transaction":            {},
					"dao.attempts.batch_insert": {},
					"taskqueue.redis.add":       {},
				}

				// wait for export
				proxyFunc := func() bool {
					resp, err := proxyClient.R().
						SetBody(`{
							"event_type": "foo.bar",
							"data": {
								"key": "value"
							}
						}`).
						SetHeader("traceparent", fmt.Sprintf("00-%s-0000000000000001-01", traceID)).
						Post("/")
					return err == nil && resp.StatusCode() == 200
				}
				assert.Eventually(GinkgoT(), proxyFunc, time.Second*5, time.Second)

				// make more tracing data
				time.Sleep(time.Second * 3)
				gotScopeNames := make(map[string]bool)
				gotSpanAttributes := make(map[string]map[string]string)
				assert.Eventually(GinkgoT(), func() bool {
					line, err := helper.FileLine(helper.OtelCollectorTracesFile, n)
					if err != nil || line == "" {
						fmt.Printf("read empty line %d", n)
						fmt.Println("")
						proxyFunc()
						return false
					}
					n++

					var trace ExportedTrace
					err = json.Unmarshal([]byte(line), &trace)
					if err != nil {
						return false
					}

					if len(trace.ResourceSpans) == 0 {
						return false
					}

					scopeNames, spanAttrs := trace.filterSpansByTraceID(traceID)
					for k, v := range scopeNames {
						gotScopeNames[k] = v
					}
					for k, v := range spanAttrs {
						gotSpanAttributes[k] = v
					}

					for _, expectedScopeName := range expectedScopeNames {
						if !gotScopeNames[expectedScopeName] {
							fmt.Printf("scope %s not exist", expectedScopeName)
							fmt.Println("")
							return false
						}
					}

					for spanName, expectedAttributes := range expectedScopeSpans {
						gotAttributes, ok := gotSpanAttributes[spanName]
						if !ok {
							fmt.Printf("span %s not exist", spanName)
							fmt.Println()
							return false
						}

						if len(expectedAttributes) > 0 {
							for k, v := range expectedAttributes {
								if _, ok := gotAttributes[k]; !ok {
									fmt.Printf("expected span %s attribute %s not exist", spanName, k)
									fmt.Println("")
									return false
								}
								valMatch := (v == "*" || reflect.DeepEqual(gotAttributes[k], v))
								if !valMatch {
									fmt.Printf("expected span %s attribute %s value not match: %s", spanName, k, v)
									fmt.Println("")
									return false
								}
							}
						}
					}
					return true
				}, time.Second*30, time.Second)
			})
		})
	}

	Context("SDK configuration by env", func() {
		var app *app.Application
		var proxyClient *resty.Client

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
			Sources:   []*entities.Source{helper.DefaultSource()},
		}
		entitiesConfig.Sources[0].Async = false

		BeforeAll(func() {
			var err error
			helper.InitOtelOutput()
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app, err = helper.Start(map[string]string{
				"WEBHOOKX_PROXY_LISTEN":                   "0.0.0.0:8081",
				"WEBHOOKX_TRACING_SAMPLING_RATE":          "1",
				"WEBHOOKX_TRACING_ATTRIBUTES":             `{"env":"test"}`,
				"WEBHOOKX_TRACING_OPENTELEMETRY_PROTOCOL": string(config.OtlpProtocolHTTP),
				"WEBHOOKX_TRACING_OPENTELEMETRY_ENDPOINT": "http://localhost:4318/v1/traces",
				"OTEL_RESOURCE_ATTRIBUTES":                "service.version=0.3",
				"OTEL_SERVICE_NAME":                       "WebhookX-Test", // env override
			})
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			n, err := helper.FileCountLine(helper.OtelCollectorTracesFile)
			assert.Nil(GinkgoT(), err)
			n++
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetBody(`{
					"event_type": "foo.bar",
					"data": {
						"key": "value"
					}
				}`).
					SetQueryParam("test", "true").
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			expected := map[string]string{"service.name": "WebhookX-Test", "service.version": "0.3", "env": "test"}
			assert.Eventually(GinkgoT(), func() bool {
				line, err := helper.FileLine(helper.OtelCollectorTracesFile, n)
				if err != nil || line == "" {
					return false
				}
				n++
				var req ExportedTrace
				_ = json.Unmarshal([]byte(line), &req)
				attributesMap := make(map[string]string)
				for _, resourceSpan := range req.ResourceSpans {
					for _, attr := range resourceSpan.Resource.Attributes {
						if attr.Value.StringValue != nil {
							attributesMap[attr.Key] = *attr.Value.StringValue
						}
					}
				}
				for name, expectVal := range expected {
					if val, ok := attributesMap[name]; !ok || val != expectVal {
						fmt.Printf("expected attribute %s not exist or value %s not match", name, val)
						fmt.Println("")
						return false
					}
				}
				return true
			}, time.Second*30, time.Second)
		})
	})
})

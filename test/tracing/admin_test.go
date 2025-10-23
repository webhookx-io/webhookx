package tracing

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"time"
)

var _ = Describe("tracing admin", Ordered, func() {
	endpoints := map[string]string{
		"http/protobuf": "http://localhost:4318/v1/traces",
		"grpc":          "localhost:4317",
	}
	for protocol, address := range endpoints {
		Context(protocol, func() {
			var app *app.Application
			var proxyClient *resty.Client
			var adminClient *resty.Client
			entitiesConfig := helper.EntitiesConfig{
				Endpoints: []*entities.Endpoint{factory.EndpointP()},
				Sources:   []*entities.Source{factory.SourceP(factory.WithSourceAsync(true))},
			}
			var gotScopeNames map[string]bool
			var gotSpanAttributes map[string]map[string]string

			BeforeAll(func() {
				helper.InitOtelOutput()
				helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()
				adminClient = helper.AdminClient()

				envs := map[string]string{
					"WEBHOOKX_TRACING_ENABLED":                "true",
					"WEBHOOKX_TRACING_SAMPLING_RATE":          "1.0",
					"WEBHOOKX_TRACING_OPENTELEMETRY_PROTOCOL": protocol,
					"WEBHOOKX_TRACING_OPENTELEMETRY_ENDPOINT": address,
				}

				app = utils.Must(helper.Start(envs))
				gotScopeNames = make(map[string]bool)
				gotSpanAttributes = make(map[string]map[string]string)
			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity", func() {
				traceID := helper.GenerateTraceID()
				expectedScopeNames := []string{
					"github.com/webhookx-io/webhookx",
					"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
				}
				entrypoint := map[string]string{
					"http.request.method":       "GET",
					"url.scheme":                "http",
					"url.path":                  "/workspaces/default/attempts",
					"http.response.status_code": "200",
					"http.response.body.size":   "*",
					"user_agent.original":       "*",
					"server.address":            "localhost",
					"server.port":               "9701",
					"network.protocol.version":  "*",
					"network.peer.address":      "*",
					"network.peer.port":         "*",
				}

				expectedScopeSpans := map[string]map[string]string{
					"api.admin":          entrypoint,
					"dao.attempts.page":  {},
					"dao.attempts.count": {},
					"dao.attempts.list":  {},
				}

				err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
				assert.NoError(GinkgoT(), err)

				n, err := helper.FileCountLine(helper.OtelCollectorTracesFile)
				assert.Nil(GinkgoT(), err)
				n++

				// make more tracing data
				for i := 0; i < 20; i++ {
					resp, err := proxyClient.R().
						SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
						Post("/")
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 200, resp.StatusCode())
				}

				assert.Eventually(GinkgoT(), func() bool {
					resp, err := adminClient.R().
						SetHeader("traceparent", fmt.Sprintf("00-%s-0000000000000001-01", traceID)).
						SetResult(api.Pagination[*entities.Attempt]{}).
						Get("/workspaces/default/attempts?page_no=1")
					result := resp.Result().(*api.Pagination[*entities.Attempt])
					return err == nil && resp.StatusCode() == 200 && len(result.Data) == 20
				}, time.Second*5, time.Second)

				time.Sleep(time.Second * 10)

				assert.Eventually(GinkgoT(), func() bool {
					line, err := helper.FileLine(helper.OtelCollectorTracesFile, n)
					if err != nil || line == "" {
						fmt.Printf("read empty line %d", n)
						fmt.Println("")
						return false
					}
					n++
					var trace ExportedTrace
					err = json.Unmarshal([]byte(line), &trace)
					if err != nil {
						fmt.Printf("unmarshal err %v", err)
						fmt.Println("")
						return false
					}

					if len(trace.ResourceSpans) == 0 {
						fmt.Printf("no resource spans")
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
								valMatch := (v == "*" || gotAttributes[k] == v)
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
})

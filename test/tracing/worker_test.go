package tracing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("tracing worker", Ordered, func() {
	endpoints := map[string]string{
		"http/protobuf": "http://localhost:4318/v1/traces",
		"grpc":          "localhost:4317",
	}
	for protocol, address := range endpoints {
		Context(protocol, func() {
			var app *app.Application
			var proxyClient *resty.Client

			BeforeAll(func() {
				helper.InitOtelOutput()
				cfg := &helper.TestEntities{
					Endpoints: []*entities.Endpoint{factory.Endpoint()},
					Sources:   []*entities.Source{factory.Source(func(o *entities.Source) { o.Async = true })},
				}
				helper.InitDB(true, cfg)
				proxyClient = helper.ProxyClient()
				envs := map[string]string{
					"WEBHOOKX_TRACING_ENABLED":                "true",
					"WEBHOOKX_TRACING_SAMPLING_RATE":          "1.0",
					"WEBHOOKX_TRACING_OPENTELEMETRY_PROTOCOL": protocol,
					"WEBHOOKX_TRACING_OPENTELEMETRY_ENDPOINT": address,
				}

				app = utils.Must(helper.Start(envs))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("sanity", func() {
				expectedScopeNames := []string{
					"github.com/webhookx-io/webhookx",
				}
				expectedScopeSpans := map[string]map[string]string{
					"worker.submit":              {},
					"worker.handle_task":         {},
					"dao.endpoints.get":          {},
					"worker.deliver":             {},
					"dao.attempt_details.insert": {},
					"taskqueue.redis.delete":     {},
				}

				n, err := helper.FileCountLine(helper.OtelCollectorTracesFile)
				assert.Nil(GinkgoT(), err)
				n++

				time.Sleep(time.Second * 3)
				// wait for export
				proxyFunc := func() bool {
					fmt.Println("send...")
					resp, err := proxyClient.R().
						SetBody(`{
							"event_type": "foo.bar",
							"data": {
								"key": "value"
							}
						}`).Post("/")
					return err == nil && resp.StatusCode() == 200
				}
				assert.Eventually(GinkgoT(), proxyFunc, time.Second*5, time.Second)

				gotScopeNames := make(map[string]bool)
				gotSpanAttributes := make(map[string]map[string]string)

				fmt.Printf("reading trace file start from line: %d\n", n)
				assert.Eventually(GinkgoT(), func() bool {
					line, err := helper.FileLine(helper.OtelCollectorTracesFile, n)
					if err != nil || line == "" {
						return false
					}
					n++

					// fmt.Printf("%s\n", line)

					var trace ExportedTrace
					err = json.Unmarshal([]byte(line), &trace)
					if err != nil {
						fmt.Printf("unmarshal err %v\n", err)
						return false
					}

					if len(trace.ResourceSpans) == 0 {
						fmt.Println("no resource spans")
						return false
					}

					// make sure worker handle full trace
					traceID := trace.getTraceIDBySpanName("worker.handle_task")
					if traceID == "" {
						fmt.Println("trace id not exist")
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
				}, time.Second*60, time.Second)
			})
		})
	}
})

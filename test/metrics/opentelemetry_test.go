package metrics

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"testing"
	"time"
)

var _ = Describe("opentelemetry", Ordered, func() {
	endpoints := map[string]string{
		"http/protobuf": "http://localhost:4318/v1/metrics",
		"grpc":          "localhost:4317",
	}

	for _, protocol := range []string{"http/protobuf", "grpc"} {
		Context(protocol, func() {
			var proxyClient *resty.Client
			var app *app.Application

			BeforeAll(func() {
				entitiesConfig := helper.EntitiesConfig{
					Endpoints: []*entities.Endpoint{factory.EndpointP(), factory.EndpointP()},
					Sources:   []*entities.Source{factory.SourceP()},
				}
				entitiesConfig.Endpoints[1].Request.Timeout = 1
				entitiesConfig.Sources[0].Async = true
				helper.InitDB(true, &entitiesConfig)
				helper.InitOtelOutput()
				proxyClient = helper.ProxyClient()
				var err error
				app, err = helper.Start(map[string]string{
					"WEBHOOKX_ADMIN_LISTEN":                   "0.0.0.0:8080",
					"WEBHOOKX_PROXY_LISTEN":                   "0.0.0.0:8081",
					"WEBHOOKX_WORKER_ENABLED":                 "true",
					"WEBHOOKX_METRICS_EXPORTS":                "opentelemetry",
					"WEBHOOKX_METRICS_PUSH_INTERVAL":          "3",
					"WEBHOOKX_METRICS_OPENTELEMETRY_PROTOCOL": protocol,
					"WEBHOOKX_METRICS_OPENTELEMETRY_ENDPOINT": endpoints[protocol],
				})
				assert.Nil(GinkgoT(), err)
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
					}`).Post("/")
					return err == nil && resp.StatusCode() == 200
				}, time.Second*5, time.Second)

				expected := []string{
					"webhookx.runtime.num_goroutine",
					"webhookx.runtime.alloc_bytes",
					"webhookx.runtime.sys_bytes",
					"webhookx.runtime.mallocs",
					"webhookx.runtime.frees",
					"webhookx.runtime.heap_objects",
					"webhookx.runtime.pause_total_ns",
					"webhookx.runtime.num_gc",

					"webhookx.request.total",
					"webhookx.request.duration",

					"webhookx.event.total",
					"webhookx.event.persisted",
					"webhookx.event.pending",

					"webhookx.attempt.total",
					"webhookx.attempt.response.duration",
					"webhookx.attempt.pending",
					"webhookx.attempt.failed",
				}

				n, err := helper.FileCountLine(helper.OtelCollectorMetricsFile)
				assert.Nil(GinkgoT(), err)
				n++
				uploaded := make(map[string]bool)
				assert.Eventually(GinkgoT(), func() bool {
					line, err := helper.FileLine(helper.OtelCollectorMetricsFile, n)
					if err != nil || line == "" {
						return false
					}
					n++
					var req ExportRequest
					_ = json.Unmarshal([]byte(line), &req)
					for _, resourceMetrics := range req.ResourceMetrics {
						for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
							for _, metrics := range scopeMetrics.Metrics {
								uploaded[metrics.Name] = true
							}
						}
					}
					for _, name := range expected {
						if !uploaded[name] {
							fmt.Println("missing metric: " + name)
							return false
						}
					}
					return true
				}, time.Second*40, time.Second)
			})
		})
	}

	Context("SDK configuration by env", func() {
		var app *app.Application

		BeforeAll(func() {
			var err error
			helper.InitOtelOutput()
			app, err = helper.Start(map[string]string{
				"WEBHOOKX_METRICS_ATTRIBUTES":             `{"env": "prod"}`,
				"WEBHOOKX_METRICS_EXPORTS":                "opentelemetry",
				"WEBHOOKX_METRICS_OPENTELEMETRY_PROTOCOL": "http/protobuf",
				"WEBHOOKX_METRICS_OPENTELEMETRY_ENDPOINT": "http://localhost:4318/v1/metrics",
				"OTEL_RESOURCE_ATTRIBUTES":                "key1=value1,key2=value2",
				"WEBHOOKX_METRICS_PUSH_INTERVAL":          "3",
			})
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			n, err := helper.FileCountLine(helper.OtelCollectorMetricsFile)
			assert.Nil(GinkgoT(), err)
			n++
			expected := []string{"service.name", "service.version", "env", "key1", "key2"}
			assert.Eventually(GinkgoT(), func() bool {
				line, err := helper.FileLine(helper.OtelCollectorMetricsFile, n)
				if err != nil || line == "" {
					return false
				}
				n++
				var req ExportRequest
				assert.Nil(GinkgoT(), json.Unmarshal([]byte(line), &req))
				attributesMap := make(map[string]bool)
				for _, resourceMetrics := range req.ResourceMetrics {
					for _, attr := range resourceMetrics.Resource.Attributes {
						attributesMap[attr.Key] = true
					}
				}
				for _, name := range expected {
					if !attributesMap[name] {
						fmt.Println("missing attribute: " + name)
						fmt.Println(line)
						return false
					}
				}
				return true
			}, time.Second*60, time.Millisecond*100)
		})
	})
})

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics opentelemetry Suite")
}

package metrics

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test"
	v1 "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

type ExportMetrics struct {
	Attributes map[string]interface{} `json:"attributes"`
	Metrics    map[string]interface{} `json:"metrics"`
}

func startMockServer(t assert.TestingT, addr string) (*http.Server, <-chan []*ExportMetrics) {
	receiver := make(chan []*ExportMetrics)

	http.HandleFunc("/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var metricsRequest v1.ExportMetricsServiceRequest
		if err := proto.Unmarshal(body, &metricsRequest); err != nil {
			log.Printf("Failed to unmarshal metrics data: %v", err)
			http.Error(w, "Failed to parse metrics data", http.StatusBadRequest)
			return
		}
		list := make([]*ExportMetrics, 0)
		defer func() {
			receiver <- list
		}()
		for _, resourceMetric := range metricsRequest.GetResourceMetrics() {
			em := &ExportMetrics{
				Attributes: make(map[string]interface{}),
				Metrics:    make(map[string]interface{}),
			}
			for _, attr := range resourceMetric.Resource.GetAttributes() {
				em.Attributes[attr.GetKey()] = attr.GetValue()
			}
			// log.Printf("Received metrics for resource: %v\n", resourceMetric.Resource)
			for _, scopeMetric := range resourceMetric.GetScopeMetrics() {
				for _, metric := range scopeMetric.Metrics {
					em.Metrics[metric.Name] = metric
				}
			}
			list = append(list, em)
		}
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr: addr,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			assert.Failf(t, "Failed to start server: %v", err.Error())
		}
	}()

	return server, receiver
}

var _ = Describe("metrics", Ordered, func() {

	Context("opentelemetry", func() {
		var app *app.Application
		var server *http.Server
		var receiver <-chan []*ExportMetrics

		BeforeAll(func() {
			server, receiver = startMockServer(GinkgoT(), ":9000")
			var err error
			app, err = test.Start(map[string]string{
				"WEBHOOKX_METRICS_EXPORTERS":              "opentelemetry",
				"WEBHOOKX_METRICS_OPENTELEMETRY_PROTOCOL": "http/protobuf",
				"WEBHOOKX_METRICS_OPENTELEMETRY_ENDPOINT": "http://localhost:9000",
			})
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
			server.Shutdown(context.Background())
		})

		BeforeEach(func() {
		})

		It("should export runtime metrics", func() {
			runtimeMetrics := map[string]bool{
				"webhookx.runtime.num_goroutine":  true,
				"webhookx.runtime.alloc_bytes":    true,
				"webhookx.runtime.sys_bytes":      true,
				"webhookx.runtime.mallocs":        true,
				"webhookx.runtime.frees":          true,
				"webhookx.runtime.heap_objects":   true,
				"webhookx.runtime.pause_total_ns": true,
				"webhookx.runtime.num_gc":         true,
			}
			assert.Eventually(GinkgoT(), func() bool {
				select {
				case metrics := <-receiver:
					for _, metric := range metrics {
						hasRuntimeMetrics := false
						for name := range runtimeMetrics {
							if _, ok := metric.Metrics[name]; !ok {
								break
							}
							hasRuntimeMetrics = true
						}
						if hasRuntimeMetrics {
							return true
						}
					}
					return false
				case <-time.After(time.Second * 10):
					return false
				}
			}, time.Second*20, time.Second)
		})

		It("worker", func() {

		})

		It("proxy", func() {

		})

	})

	Context("datadog", func() {
	})

})

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

package tracing

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/outbound"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var _ = Describe("tracing worker", Ordered, func() {

	plugin.RegisterPlugin(plugin.TypeOutbound, "outbound", func() plugin.Plugin { return &outbound.OutboundPlugin{} })

	endpoints := map[string]string{
		"grpc":          "localhost:4317",
		"http/protobuf": "http://localhost:4318/v1/traces",
	}
	for _, protocol := range []string{"grpc", "http/protobuf"} {
		Context(protocol, func() {
			var exportor = tracetest.NewInMemoryExporter()
			var spanProcessor = sdktrace.NewSimpleSpanProcessor(exportor)
			var app *app.Application
			var proxyClient *resty.Client
			var db *db.DB

			entitiesConfig := helper.TestEntities{
				Endpoints: []*entities.Endpoint{factory.Endpoint(factory.WithEndpointPlugins(
					factory.Plugin("outbound", factory.WithPluginConfig(outbound.Config{}))))},
				Sources: []*entities.Source{factory.Source()},
			}

			BeforeAll(func() {
				db = helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()

				envs := map[string]string{
					"WEBHOOKX_TRACING_INSTRUMENTATIONS":       "@all",
					"WEBHOOKX_TRACING_SAMPLING_RATE":          "1.0",
					"WEBHOOKX_TRACING_OPENTELEMETRY_PROTOCOL": protocol,
					"WEBHOOKX_TRACING_OPENTELEMETRY_ENDPOINT": endpoints[protocol],
				}
				app = helper.MustStart(envs)

				tp := (tracing.GetTracer().TracerProvider).(*sdktrace.TracerProvider)
				tp.RegisterSpanProcessor(spanProcessor)
			})

			AfterAll(func() {
				tp := (tracing.GetTracer().TracerProvider).(*sdktrace.TracerProvider)
				tp.UnregisterSpanProcessor(spanProcessor)
				app.Stop()
			})

			It("sanity", func() {
				resp, err := proxyClient.R().
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				eventId := resp.Header().Get(constants.HeaderEventId)

				assert.Eventually(GinkgoT(), func() bool {
					q := query.AttemptQuery{}
					q.EventId = &eventId
					list, err := db.Attempts.List(context.TODO(), &q)
					if err != nil || len(list) == 0 {
						return false
					}
					return list[0].Status == entities.AttemptStatusSuccess
				}, time.Second*3, time.Second)

				assertor := NewTraceAsserter(exportor.GetSpans().Snapshots())
				err = assertor.AssertSpans(map[string]map[string]string{
					"worker.task.submit":         {},
					"worker.task.run":            {},
					"dao.endpoints.get":          {},
					"plugin.outbound.outbound":  {},
					"http.send":                  {},
					"queue.request_log.add":      {},
					"dao.attempts.update_result": {},
					"task_queue.redis.delete":    {},
				})
				assert.NoError(GinkgoT(), err)
			})
		})
	}
})

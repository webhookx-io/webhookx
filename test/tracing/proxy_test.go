package tracing

import (
	"context"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/inbound"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var _ = Describe("tracing proxy", Ordered, func() {

	plugin.RegisterPlugin(plugin.TypeInbound, "inbound", func() plugin.Plugin { return &inbound.InboundPlugin{} })

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

			entitiesConfig := helper.TestEntities{
				Endpoints: []*entities.Endpoint{factory.Endpoint()},
				Sources: []*entities.Source{
					factory.Source(factory.WithSourcePlugins(
						factory.Plugin("inbound", factory.WithPluginConfig(inbound.Config{})))),
					factory.Source(func(o *entities.Source) {
						o.Config.HTTP.Path = "/async"
						o.Async = true
					}),
				},
			}

			BeforeAll(func() {
				helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()
				tr := otelhttp.NewTransport(http.DefaultTransport)
				proxyClient.SetTransport(tr)

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
				ctx, span := tracing.Start(context.TODO(), "test")
				defer span.End()

				resp, err := proxyClient.R().
					SetContext(ctx).
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())

				assertor := NewTraceAsserter(exportor.GetSpans().Snapshots())
				assertor.FilterTraceID(trace.SpanContextFromContext(ctx).TraceID().String())
				err = assertor.AssertSpans(map[string]map[string]string{
					"request": {
						"http.request.method":       "POST",
						"url.scheme":                "http",
						"url.path":                  "/",
						"http.response.status_code": "200",
						"http.request.body.size":    "*",
						"http.response.body.size":   "*",
						"user_agent.original":       "*",
						"server.address":            "localhost",
						"server.port":               "9700",
						"network.protocol.version":  "*",
						"network.peer.address":      "*",
						"network.peer.port":         "*",
					},
					"resolve_source":             {},
					"plugin.inbound.inbound":     {},
					"event.ingest":               {},
					"event.fanout":               {},
					"db.transaction":             {},
					"attempt.schedule":           {},
					"task_queue.redis.add":       {},
					"dao.attempts.update_status": {},
				})
				assert.NoError(GinkgoT(), err)
			})

			It("span queue.messages.process should contain links", func() {
				ctx, span := tracing.Start(context.TODO(), "test")
				defer span.End()
				resp, err := proxyClient.R().
					SetContext(ctx).
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/async")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())

				assert.Eventually(GinkgoT(), func() bool {
					traceId := trace.SpanContextFromContext(ctx).TraceID()
					assertor := NewTraceAsserter(exportor.GetSpans().Snapshots())
					s := assertor.FindSpan("queue.messages.process")
					return len(s.Links()) > 0 && s.Links()[0].SpanContext.TraceID().String() == traceId.String()
				}, time.Second*3, time.Second)
			})
		})
	}
})

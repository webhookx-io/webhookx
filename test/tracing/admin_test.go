package tracing

import (
	"context"
	"net/http"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/test/helper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var _ = Describe("tracing admin", Ordered, func() {
	endpoints := map[string]string{
		"http/protobuf": "http://localhost:4318/v1/traces",
		"grpc":          "localhost:4317",
	}
	for _, protocol := range []string{"grpc", "http/protobuf"} {
		Context(protocol, func() {
			var exportor = tracetest.NewInMemoryExporter()
			var spanProcessor = sdktrace.NewSimpleSpanProcessor(exportor)
			var app *app.Application
			var adminClient *resty.Client

			BeforeAll(func() {
				adminClient = helper.AdminClient()
				tr := otelhttp.NewTransport(http.DefaultTransport)
				adminClient.SetTransport(tr)

				app = helper.MustStart(map[string]string{
					"WEBHOOKX_TRACING_INSTRUMENTATIONS":       "@all",
					"WEBHOOKX_TRACING_SAMPLING_RATE":          "1.0",
					"WEBHOOKX_TRACING_OPENTELEMETRY_PROTOCOL": protocol,
					"WEBHOOKX_TRACING_OPENTELEMETRY_ENDPOINT": endpoints[protocol],
				})

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

				resp, err := adminClient.R().
					SetContext(ctx).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())

				assertor := NewTraceAsserter(exportor.GetSpans().Snapshots())
				assertor.FilterTraceID(trace.SpanContextFromContext(ctx).TraceID().String())
				err = assertor.AssertSpans(map[string]map[string]string{
					"admin.request": {
						"http.request.method":       "GET",
						"url.scheme":                "http",
						"url.path":                  "/workspaces/default/endpoints",
						"http.response.status_code": "200",
						"http.response.body.size":   "*",
						"user_agent.original":       "*",
						"server.address":            "localhost",
						"server.port":               "9701",
						"network.protocol.version":  "*",
						"network.peer.address":      "*",
						"network.peer.port":         "*",
					},
					"admin.endpoints.page": {},
					"dao.endpoints.page":   {},
					"dao.endpoints.count":  {},
					"dao.endpoints.list":   {},
				})
				assert.NoError(GinkgoT(), err)

				// validate resources
				testSpan := assertor.FindSpan("admin.request")
				assert.NotNil(GinkgoT(), testSpan)
				err = assertor.AssertAttributes(testSpan.Resource().Attributes(), map[string]string{
					"service.name":        "webhookx",
					"service.instance.id": "*",
					"service.version":     webhookx.VERSION,
				})
				assert.NoError(GinkgoT(), err)
			})
		})
	}
})

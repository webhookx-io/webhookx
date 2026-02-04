package tracing

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/test/helper"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var _ = Describe("tracing", Ordered, func() {

	Context("attributes", func() {
		var exportor = tracetest.NewInMemoryExporter()
		var spanProcessor = sdktrace.NewSimpleSpanProcessor(exportor)
		var app *app.Application

		BeforeAll(func() {
			app = helper.MustStart(map[string]string{
				"WEBHOOKX_TRACING_INSTRUMENTATIONS": "@all",
				"WEBHOOKX_TRACING_ATTRIBUTES":       `{"env":"test"}`,
			})
			tp := (tracing.GetTracer().TracerProvider).(*sdktrace.TracerProvider)
			tp.RegisterSpanProcessor(spanProcessor)
		})

		AfterAll(func() {
			tp := (tracing.GetTracer().TracerProvider).(*sdktrace.TracerProvider)
			tp.UnregisterSpanProcessor(spanProcessor)
			app.Stop()
		})

		It("span should contain custom attribute", func() {
			_, span := tracing.Start(context.Background(), "test")
			span.End()
			assertor := NewTraceAsserter(exportor.GetSpans().Snapshots())
			s := assertor.FindSpan("test")
			assert.NotNil(GinkgoT(), s)
			err := assertor.AssertAttributes(s.Resource().Attributes(), map[string]string{
				"env": "test",
			})
			assert.NoError(GinkgoT(), err)
		})
	})

	Context("disabled", func() {
		var app *app.Application

		BeforeAll(func() {
			app = helper.MustStart(nil)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("tracing should be disabled ", func() {
			ctx := context.Background()
			tCtx, span := tracing.Start(ctx, "test")
			spanCtxValid := span.SpanContext().IsValid()
			assert.False(GinkgoT(), spanCtxValid, "span context should be invalid")
			valid := trace.SpanContextFromContext(tCtx).IsValid()
			assert.False(GinkgoT(), valid, "span context should be invalid")
		})
	})

	Context("configured via OTEL environments", func() {
		var exportor = tracetest.NewInMemoryExporter()
		var spanProcessor = sdktrace.NewSimpleSpanProcessor(exportor)
		var app *app.Application
		var cancel func()

		BeforeAll(func() {
			cancel = helper.SetEnvs(map[string]string{
				"OTEL_RESOURCE_ATTRIBUTES": "service.version=0.0.1",
				"OTEL_SERVICE_NAME":        "WebhookX-Test",
			})
			app = helper.MustStart(map[string]string{
				"WEBHOOKX_TRACING_INSTRUMENTATIONS": "@all",
				"WEBHOOKX_TRACING_SAMPLING_RATE":    "1",
				"WEBHOOKX_TRACING_ATTRIBUTES":       `{"env":"test"}`,
			})
			tp := (tracing.GetTracer().TracerProvider).(*sdktrace.TracerProvider)
			tp.RegisterSpanProcessor(spanProcessor)
		})

		AfterAll(func() {
			cancel()
			tp := (tracing.GetTracer().TracerProvider).(*sdktrace.TracerProvider)
			tp.UnregisterSpanProcessor(spanProcessor)
			app.Stop()
		})

		It("otel environment should be applied", func() {
			_, span := tracing.Start(context.Background(), "test")
			span.End()
			assertor := NewTraceAsserter(exportor.GetSpans().Snapshots())
			s := assertor.FindSpan("test")
			assert.NotNil(GinkgoT(), s)
			err := assertor.AssertAttributes(s.Resource().Attributes(), map[string]string{
				"env":                 "test",
				"service.name":        "WebhookX-Test",
				"service.version":     "0.0.1",
				"service.instance.id": "*",
			})
			assert.NoError(GinkgoT(), err)
		})
	})
})

func TestTracing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tracing Suite")
}

package tracing

import (
	"context"
	"sync/atomic"

	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/utils"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	NoopTracer = NewTracer(noop.NewTracerProvider(), nil)
)

var (
	globalTracer atomic.Pointer[Tracer]
)

func init() {
	globalTracer.Store(NoopTracer)
}

func Init(conf *modules.TracingConfig) error {
	if !conf.Enabled() {
		return nil
	}

	tr, err := SetupOTEL(conf)
	if err != nil {
		return err
	}

	globalTracer.Store(NewTracer(tr, conf.Instrumentations))
	return nil
}

type Tracer struct {
	instrumented map[string]bool
	trace.TracerProvider
}

func NewTracer(tp trace.TracerProvider, instrumentations []string) *Tracer {
	presets := map[string][]string{
		"@all": {"request", "plugin", "dao"},
	}
	instrumentations = utils.ResolveAlias(presets, instrumentations)
	instrumented := make(map[string]bool)
	for _, name := range instrumentations {
		instrumented[name] = true
	}
	return &Tracer{
		instrumented:   instrumented,
		TracerProvider: tp,
	}
}

func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := t.Tracer(instrumentationName)
	return tracer.Start(ctx, spanName, opts...)
}

func (t *Tracer) Stop(ctx context.Context) error {
	if pr, ok := t.TracerProvider.(*sdktrace.TracerProvider); ok {
		return pr.Shutdown(ctx)
	}
	return nil
}

func (t *Tracer) Enabled(name string) bool {
	return t.instrumented[name]
}

func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return globalTracer.Load().Start(ctx, spanName, opts...)
}

func Enabled(name string) bool {
	return globalTracer.Load().Enabled(name)
}

func GetTracer() *Tracer {
	return globalTracer.Load()
}

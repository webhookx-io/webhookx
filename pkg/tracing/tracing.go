package tracing

import (
	"context"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func New(conf *modules.TracingConfig) (*Tracer, error) {
	if !conf.Enabled {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return nil, nil
	}

	tr, err := SetupOTEL(conf)
	if err != nil {
		return nil, err
	}

	return NewTracer(tr), nil
}

func TracerFromContext(ctx context.Context) trace.Tracer {
	var tp trace.TracerProvider
	if !trace.SpanContextFromContext(ctx).IsValid() {
		tp = otel.GetTracerProvider()
	} else {
		tp = trace.SpanFromContext(ctx).TracerProvider()
	}

	return tp.Tracer(instrumentationName)
}

type Tracer struct {
	trace.TracerProvider
}

func NewTracer(tracerProvider trace.TracerProvider) *Tracer {
	return &Tracer{
		TracerProvider: tracerProvider,
	}

}

func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	tracer := t.Tracer(instrumentationName)
	spanCtx, span := tracer.Start(ctx, spanName, opts...)
	return spanCtx, span
}

func (t *Tracer) Stop() error {
	if t == nil {
		return nil
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	if pr, ok := t.TracerProvider.(*sdktrace.TracerProvider); ok {
		return pr.Shutdown(ctx)
	}
	return nil
}

func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := TracerFromContext(ctx)
	return tracer.Start(ctx, spanName, opts...)
}

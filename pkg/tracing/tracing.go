package tracing

import (
	"context"
	"github.com/webhookx-io/webhookx/config"
	"go.opentelemetry.io/otel/trace"
	"io"
)

var internalTracer *Tracer

func New(conf *config.TracingConfig) (*Tracer, error) {
	if !conf.Enabled {
		return nil, nil
	}

	tr, closer, err := SetupOTEL(conf)
	if err != nil {
		return nil, err
	}

	internalTracer = NewTracer(tr, closer)
	return internalTracer, nil
}

func TracerFromContext(ctx context.Context) *Tracer {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return nil
	}

	span := trace.SpanFromContext(ctx)
	if span != nil && span.TracerProvider() != nil {
		tracerProvider := span.TracerProvider()
		tracer := tracerProvider.Tracer(instrumentationName)
		tr, ok := tracer.(*Tracer)
		if ok {
			return tr
		} else {
			if internalTracer == nil {
				internalTracer = NewTracer(tracerProvider, &tpCloser{tracerProvider})
			}
			return internalTracer
		}
	}

	return nil
}

type Span struct {
	trace.Span

	tracerProvider *TracerProvider
}

func (s Span) TracerProvider() trace.TracerProvider {
	return s.tracerProvider
}

type TracerProvider struct {
	trace.TracerProvider

	tracer *Tracer
}

func (t TracerProvider) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	if name == instrumentationName {
		return t.tracer
	}

	return t.TracerProvider.Tracer(name, options...)
}

type Tracer struct {
	trace.Tracer
	io.Closer
}

func NewTracer(tracerProvider trace.TracerProvider, closer io.Closer) *Tracer {
	return &Tracer{
		Tracer: tracerProvider.Tracer(instrumentationName),
		Closer: closer,
	}
}

func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t == nil {
		return ctx, nil
	}

	spanCtx, span := t.Tracer.Start(ctx, spanName, opts...)

	wrappedSpan := &Span{Span: span, tracerProvider: &TracerProvider{tracer: t}}

	return trace.ContextWithSpan(spanCtx, wrappedSpan), wrappedSpan
}

func (t *Tracer) Stop() error {
	if t == nil {
		return nil
	}
	if t.Closer != nil {
		return t.Closer.Close()
	}
	return nil
}

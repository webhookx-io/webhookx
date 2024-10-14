package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type entrypointTracing struct {
	tracer *tracing.Tracer

	entrypoint string
	next       http.Handler
}

// WrapEntryPointHandler Wraps tracing to alice.Constructor.
func WrapEntryPointHandler(ctx context.Context, tracer *tracing.Tracer, entryPointName string) alice.Constructor {
	return func(next http.Handler) http.Handler {
		if tracer == nil {
			tracer = tracing.NewTracer(noop.Tracer{}, nil, nil, nil)
		}

		return newEntrypoint(ctx, tracer, entryPointName, next)
	}
}

// newEntrypoint creates a new tracing middleware for incoming requests.
func newEntrypoint(ctx context.Context, tracer *tracing.Tracer, entryPointName string, next http.Handler) http.Handler {
	zap.S().Debugf("new entrypoint %s", entryPointName)

	if tracer == nil {
		tracer = tracing.NewTracer(noop.Tracer{}, nil, nil, nil)
	}

	return &entrypointTracing{
		entrypoint: entryPointName,
		tracer:     tracer,
		next:       next,
	}
}

func (e *entrypointTracing) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	tracingCtx := tracing.ExtractCarrierIntoContext(req.Context(), req.Header)
	start := time.Now()
	tracingCtx, span := e.tracer.Start(tracingCtx, "entrypoint", trace.WithSpanKind(trace.SpanKindServer), trace.WithTimestamp(start))

	req = req.WithContext(tracingCtx)

	span.SetAttributes(attribute.String("entrypoint", e.entrypoint))

	e.tracer.CaptureServerRequest(span, req)

	recorder := newStatusCodeRecorder(rw, http.StatusOK)
	e.next.ServeHTTP(recorder, req)

	e.tracer.CaptureResponse(span, recorder.Header(), recorder.Status(), trace.SpanKindServer)

	end := time.Now()
	span.End(trace.WithTimestamp(end))

}

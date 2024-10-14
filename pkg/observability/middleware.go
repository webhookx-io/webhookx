package observability

import (
	"context"
	"net/http"

	"github.com/justinas/alice"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Traceable embeds tracing information.
type Traceable interface {
	GetTracingInformation() (name string, typeName string, spanKind trace.SpanKind)
}

// WrapMiddleware adds traceability to an alice.Constructor.
func WrapMiddleware(ctx context.Context, constructor alice.Constructor) alice.Constructor {
	return func(next http.Handler) http.Handler {
		if constructor == nil {
			return nil
		}
		handler := constructor(next)

		if traceableHandler, ok := handler.(Traceable); ok {
			name, typeName, spanKind := traceableHandler.GetTracingInformation()
			zap.S().Debugw("new middleware", "tracing", name)
			return NewMiddleware(handler, name, typeName, spanKind)
		}
		return handler
	}
}

// NewMiddleware returns a http.Handler struct.
func NewMiddleware(next http.Handler, name string, typeName string, spanKind trace.SpanKind) http.Handler {
	return &middlewareTracing{
		next:     next,
		name:     name,
		typeName: typeName,
		spanKind: spanKind,
	}
}

// middlewareTracing is used to wrap http handler middleware.
type middlewareTracing struct {
	next     http.Handler
	name     string
	typeName string
	spanKind trace.SpanKind
}

func (w *middlewareTracing) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if tracer := tracing.TracerFromContext(req.Context()); tracer != nil {
		tracingCtx, span := tracer.Start(req.Context(), w.typeName, trace.WithSpanKind(w.spanKind))
		defer span.End()

		req = req.WithContext(tracingCtx)

		span.SetAttributes(attribute.String("webhookx.middleware.name", w.name))
	}

	if w.next != nil {
		w.next.ServeHTTP(rw, req)
	}
}

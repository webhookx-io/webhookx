package observability

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
)

type wrapper struct {
	name string
	rt   http.RoundTripper
}

func NewObservabilityRoundTripper(name string, rt http.RoundTripper) http.RoundTripper {
	return &wrapper{
		name: name,
		rt:   rt,
	}
}

func (t *wrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	var span trace.Span
	var tracingCtx context.Context
	var tracer *tracing.Tracer
	if tracer = tracing.TracerFromContext(req.Context()); tracer != nil {
		tracingCtx, span = tracer.Start(req.Context(), t.name, trace.WithSpanKind(trace.SpanKindClient), trace.WithTimestamp(start))
		defer span.End()

		req = req.WithContext(tracingCtx)

		tracer.CaptureClientRequest(span, req)
		tracing.InjectContextIntoCarrier(req)
	}

	var statusCode int
	var headers http.Header
	response, err := t.rt.RoundTrip(req)
	if err != nil {
		statusCode = ComputeStatusCode(err)
	}
	if response != nil {
		statusCode = response.StatusCode
		headers = response.Header
	}

	if tracer != nil {
		tracer.CaptureResponse(span, headers, statusCode, trace.SpanKindClient)
	}

	end := time.Now()

	// Ending the span as soon as the response is handled
	// If any errors happen earlier, this span will be close by the defer instruction.
	if span != nil {
		span.End(trace.WithTimestamp(end))
	}

	return response, err
}

// StatusClientClosedRequest non-standard HTTP status code for client disconnection.
const StatusClientClosedRequest = 499

// ComputeStatusCode computes the HTTP status code according to the given error.
func ComputeStatusCode(err error) int {
	switch {
	case errors.Is(err, io.EOF):
		return http.StatusBadGateway
	case errors.Is(err, context.Canceled):
		return StatusClientClosedRequest
	default:
		var netErr net.Error
		if errors.As(err, &netErr) {
			if netErr.Timeout() {
				return http.StatusGatewayTimeout
			}

			return http.StatusBadGateway
		}
	}

	return http.StatusInternalServerError
}

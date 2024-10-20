package opentelemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"

	"github.com/webhookx-io/webhookx/config"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/encoding/gzip"
)

const TracerName = "github.com/webhookx-io/webhookx"

type OpentelemetryConfig config.OpenTelemetryConfig

func (o *OpentelemetryConfig) Setup(serviceName string, samplingRate float64, globalAttributes map[string]string) (trace.Tracer, io.Closer, error) {
	var err error
	var exporter *otlptrace.Exporter

	zap.S().Infof("Setting up OpenTelemetry exporter %w", o.GRPC)
	if o.HTTP.Endpoint != "" {
		exporter, err = setupHTTPExporter(o.HTTP)
	} else if o.GRPC.Endpoint != "" {
		exporter, err = setupGRPCExporter(o.GRPC)
	} else {
		zap.S().Info("No exporter configured, skipping setup")
		return nil, nil, errors.New("exporter is not configured, failed to setup")
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup exporter: %w", err)
	}

	attr := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(config.VERSION),
	}

	for k, v := range globalAttributes {
		attr = append(attr, attribute.String(k, v))
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(attr...), // Add custom attributes
		resource.WithFromEnv(),           // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(),      // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),           // Discover and provide process information.
		resource.WithOS(),                // Discover and provide OS information.
		resource.WithContainer(),         // Discover and provide container information.
		resource.WithHost(),              // Discover and provide host information.
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build resource: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplingRate)),
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())
	return tracerProvider.Tracer(TracerName), &tpCloser{provider: tracerProvider}, err
}

func setupHTTPExporter(c config.OtelEndpoint) (*otlptrace.Exporter, error) {
	endpoint, err := url.Parse(c.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid collector endpoint %q: %w", c.Endpoint, err)
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint.Host),
		otlptracehttp.WithHeaders(c.Headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}

	if endpoint.Scheme == "http" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if endpoint.Path != "" {
		opts = append(opts, otlptracehttp.WithURLPath(endpoint.Path))
	}

	return otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
}

func setupGRPCExporter(cfg config.OtelEndpoint) (*otlptrace.Exporter, error) {
	host, port, err := net.SplitHostPort(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid collector endpoint %q: %w", cfg.Endpoint, err)
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%s", host, port)),
		otlptracegrpc.WithHeaders(cfg.Headers),
		otlptracegrpc.WithCompressor(gzip.Name),
	}

	return otlptrace.New(context.Background(), otlptracegrpc.NewClient(opts...))
}

// tpCloser converts a TraceProvider into an io.Closer.
type tpCloser struct {
	provider *sdktrace.TracerProvider
}

func (t *tpCloser) Close() error {
	if t == nil {
		return nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	return t.provider.Shutdown(ctx)
}

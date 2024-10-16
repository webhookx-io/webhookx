package metrics

import (
	"context"
	"fmt"
	"github.com/webhookx-io/webhookx/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/semconv/v1.26.0"
	"time"
)

func newHTTPExporter(cfg *config.OTLPHttp) (metric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(cfg.Endpoint),
		otlpmetrichttp.WithHeaders(cfg.Headers),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
	}
	return otlpmetrichttp.New(context.Background(), opts...)
}

func newGRPCExporter(cfg *config.OTLPgRPC) (metric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpointURL(cfg.Endpoint),
		//otlpmetricgrpc.WithHeaders(cfg.Headers),
		//otlpmetricgrpc.WithCompressor(gzip.Name),
	}

	if cfg.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	//if config.TLS != nil {
	//	tlsConfig, err := config.TLS.CreateTLSConfig(ctx)
	//	if err != nil {
	//		return nil, fmt.Errorf("creating TLS client config: %w", err)
	//	}
	//
	//	opts = append(opts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	//}

	return otlpmetricgrpc.New(context.Background(), opts...)
}

func SetupOpentelemetry(ctx context.Context, cfg *config.Opentelemetry, metrics *Metrics) error {
	var err error
	var exporter metric.Exporter

	//if cfg.HTTP != nil {
	//	exporter, err = newHTTPExporter(cfg.HTTP)
	//} else {
	exporter, err = newGRPCExporter(cfg.GRPC)
	//}

	if err != nil {
		return fmt.Errorf("failed to setup exporter: %w", err)
	}

	attr := []attribute.KeyValue{
		semconv.ServiceNameKey.String("webhookx"),
		semconv.ServiceVersionKey.String(config.VERSION),
	}
	res, err := resource.New(
		ctx,
		resource.WithAttributes(attr...), // Add custom reso
		// TODO
		resource.WithFromEnv(),      // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(), // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),      // Discover and provide process information.
		resource.WithOS(),           // Discover and provide OS information.
		resource.WithContainer(),    // Discover and provide container information.
		resource.WithHost(),         // Discover and provide host information.
	)
	if err != nil {
		return fmt.Errorf("failed to build resource: %w", err)
	}

	opts := []metric.PeriodicReaderOption{
		metric.WithInterval(time.Second * time.Duration(cfg.Interval)),
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter, opts...)),
		// View to customize histogram buckets and rename a single histogram instrument.
		//metric.WithView(metric.NewView(
		//	sdkmetric.Instrument{Name: "traefik_*_request_duration_seconds"},
		//	sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
		//		Boundaries: cfg.ExplicitBoundaries,
		//	}},
		//)),
	)

	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter("github.com/webhookx-io/webhookx")

	prefix := "webhookx."

	// proxy
	metrics.RequestCount = NewCounter(meter, prefix+"request.total", "todo")
	metrics.RequestDuration = NewHistogram(meter, prefix+"request.duration", "todo", "s")

	// runtime
	metrics.Runtime.Goroutine = NewGauge(meter, prefix+"runtime.goroutine", "todo")
	metrics.Runtime.Alloc = NewGauge(meter, prefix+"runtime.mem_stats.alloc", "todo")
	metrics.Runtime.TotalAlloc = NewGauge(meter, prefix+"runtime.mem_stats.total_alloc", "todo")
	metrics.Runtime.Sys = NewGauge(meter, prefix+"runtime.mem_stats.sys", "todo")

	metrics.TaskCount = NewCounter(meter, prefix+"taskqueue.total", "todo")
	metrics.TaskQueueSize = NewGauge(meter, prefix+"taskqueue.size", "todo")
	metrics.TaskQueueConsumeTotal = NewCounter(meter, prefix+"taskqueue.consume.total", "todo")

	metrics.EventPersistCount = NewCounter(meter, prefix+"event_persiste.total", "todo")

	metrics.ProxyQueueSize = NewGauge(meter, prefix+"queue.size", "todo")

	// worker
	metrics.AttemptsTotal = NewCounter(meter, prefix+"attempts.total", "todo")
	metrics.AttemptsFailed = NewCounter(meter, prefix+"attempts_failed.total", "todo")
	metrics.AttemptsResponseDuration = NewHistogram(meter, prefix+"attempts.response.duration", "todo", "s")

	return nil
}

package metrics

import (
	"context"
	"fmt"

	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/config/modules"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"
)

const (
	prefix = "webhookx."
)

func newHTTPExporter(endpoint string) (metric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(endpoint),
	}
	return otlpmetrichttp.New(context.Background(), opts...)
}

func newGRPCExporter(endpoint string) (metric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	}
	return otlpmetricgrpc.New(context.Background(), opts...)
}

func SetupOpentelemetry(attributes map[string]string, cfg modules.OpentelemetryMetrics, metrics *Metrics) error {
	var err error
	var exporter metric.Exporter
	switch cfg.Protocol {
	case modules.OtlpProtocolHTTP:
		exporter, err = newHTTPExporter(cfg.Endpoint)
	case modules.OtlpProtocolGRPC:
		exporter, err = newGRPCExporter(cfg.Endpoint)
	}
	if err != nil {
		return fmt.Errorf("failed to setup exporter: %v", err)
	}

	// custom attributes
	attrs := make([]attribute.KeyValue, len(attributes))
	for name, value := range attributes {
		attrs = append(attrs, attribute.String(name, value))
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(semconv.ServiceNameKey.String("webhookx")),
		resource.WithAttributes(semconv.ServiceVersionKey.String(webhookx.VERSION)),
		resource.WithFromEnv(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return fmt.Errorf("failed to build resource: %w", err)
	}

	opts := []metric.PeriodicReaderOption{
		metric.WithInterval(metrics.Interval),
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter, opts...)),
	)
	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter("github.com/webhookx-io/webhookx")

	// proxy metrics
	metrics.ProxyRequestCounter = NewCounter(meter, prefix+"request.total", "")
	metrics.ProxyRequestDurationHistogram = NewHistogram(meter, prefix+"request.duration", "", "s")

	// runtime metrics
	metrics.RuntimeGoroutine = NewGauge(meter, prefix+"runtime.num_goroutine", "")
	metrics.RuntimeAlloc = NewGauge(meter, prefix+"runtime.alloc_bytes", "")
	metrics.RuntimeSys = NewGauge(meter, prefix+"runtime.sys_bytes", "")
	metrics.RuntimeMallocs = NewGauge(meter, prefix+"runtime.mallocs", "")
	metrics.RuntimeFrees = NewGauge(meter, prefix+"runtime.frees", "")
	metrics.RuntimeHeapObjects = NewGauge(meter, prefix+"runtime.heap_objects", "")
	metrics.RuntimePauseTotalNs = NewGauge(meter, prefix+"runtime.pause_total_ns", "")
	metrics.RuntimeGC = NewGauge(meter, prefix+"runtime.num_gc", "")

	// worker metrics
	metrics.AttemptTotalCounter = NewCounter(meter, prefix+"attempt.total", "")
	metrics.AttemptFailedCounter = NewCounter(meter, prefix+"attempt.failed", "")
	metrics.AttemptPendingGauge = NewGauge(meter, prefix+"attempt.pending", "")
	metrics.AttemptResponseDurationHistogram = NewHistogram(meter, prefix+"attempt.response.duration", "", "s")

	// event metrics
	metrics.EventTotalCounter = NewCounter(meter, prefix+"event.total", "")
	metrics.EventPersistCounter = NewCounter(meter, prefix+"event.persisted", "")
	metrics.EventPendingGauge = NewGauge(meter, prefix+"event.pending", "")

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) { zap.S().Error(err) }))

	return nil
}

func StopOpentelemetry(ctx context.Context) error {
	return otel.GetMeterProvider().(*metric.MeterProvider).Shutdown(ctx)
}

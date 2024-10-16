package metrics

import (
	"context"
	"github.com/go-kit/kit/metrics"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"runtime"
	"time"
)

type Metrics struct {
	ctx    context.Context
	cancel context.CancelFunc

	Runtime RuntimeMetrics

	RequestCount    metrics.Counter
	RequestDuration metrics.Histogram

	TaskCount             metrics.Counter
	TaskQueueSize         metrics.Gauge
	TaskQueueConsumeTotal metrics.Counter

	EventPersistCount metrics.Counter

	ProxyQueueSize metrics.Gauge

	AttemptsTotal            metrics.Counter
	AttemptsFailed           metrics.Counter
	AttemptsResponseDuration metrics.Histogram
}

type RuntimeMetrics struct {
	Goroutine  metrics.Gauge
	Alloc      metrics.Gauge
	TotalAlloc metrics.Gauge
	Sys        metrics.Gauge
}

func (m *Metrics) Stop() error {
	m.cancel()
	return nil
}

func New(cfg config.MetricsConfig) (*Metrics, error) {
	metrics := &Metrics{}

	metrics.ctx, metrics.cancel = context.WithCancel(context.Background())

	//if cfg.Datadog != nil {
	//	err := SetupDataDog(metrics.ctx, cfg.Datadog, metrics)
	//	if err != nil {
	//		return nil, err
	//	}
	//}
	if cfg.OTLP != nil {
		err := SetupOpentelemetry(metrics.ctx, cfg.OTLP, metrics)
		if err != nil {
			return nil, err
		}
	}

	schedule.Schedule(metrics.ctx, func() {
		metrics.Runtime.Goroutine.Set(float64(runtime.NumGoroutine()))
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		//fmt.Printf("%d %d %d %d\n", m.Alloc, m.TotalAlloc, m.NumGC, m.Sys)
		metrics.Runtime.Alloc.Set(float64(m.Alloc))
		metrics.Runtime.TotalAlloc.Set(float64(m.TotalAlloc))
		metrics.Runtime.Sys.Set(float64(m.Sys))
	}, time.Second)

	return metrics, nil
}

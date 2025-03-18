package metrics

import (
	"context"
	"github.com/go-kit/kit/metrics"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"go.uber.org/zap"
	"runtime"
	"time"
)

type Metrics struct {
	ctx    context.Context
	cancel context.CancelFunc

	Enabled  bool
	Interval time.Duration

	// runtime metrics

	RuntimeGoroutine    metrics.Gauge
	RuntimeAlloc        metrics.Gauge
	RuntimeSys          metrics.Gauge
	RuntimeMallocs      metrics.Gauge
	RuntimeFrees        metrics.Gauge
	RuntimeHeapObjects  metrics.Gauge
	RuntimePauseTotalNs metrics.Gauge
	RuntimeGC           metrics.Gauge

	// worker metrics

	AttemptTotalCounter              metrics.Counter
	AttemptFailedCounter             metrics.Counter
	AttemptPendingGauge              metrics.Gauge
	AttemptResponseDurationHistogram metrics.Histogram

	// proxy metrics

	ProxyRequestCounter           metrics.Counter
	ProxyRequestDurationHistogram metrics.Histogram

	// events metrics
	EventTotalCounter   metrics.Counter
	EventPersistCounter metrics.Counter
	EventPendingGauge   metrics.Gauge
}

func (m *Metrics) Stop() error {
	m.cancel()
	if m.Enabled {
		return StopOpentelemetry()
	}
	return nil
}

func New(cfg config.MetricsConfig) (*Metrics, error) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Metrics{
		ctx:     ctx,
		cancel:  cancel,
		Enabled: len(cfg.Exports) > 0,
	}

	if len(cfg.Exports) > 0 {
		m.Interval = time.Second * time.Duration(cfg.PushInterval)
		err := SetupOpentelemetry(cfg.Attributes, cfg.Opentelemetry, m)
		if err != nil {
			return nil, err
		}
		schedule.Schedule(m.ctx, m.collectRuntimeStats, m.Interval)
		zap.S().Infof("enabled metric exports: %v", cfg.Exports)
	}

	return m, nil
}

func (m *Metrics) collectRuntimeStats() {
	m.RuntimeGoroutine.Set(float64(runtime.NumGoroutine()))

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	m.RuntimeAlloc.Set(float64(stats.Alloc))
	m.RuntimeSys.Set(float64(stats.Sys))
	m.RuntimeMallocs.Set(float64(stats.Mallocs))
	m.RuntimeFrees.Set(float64(stats.Frees))
	m.RuntimeHeapObjects.Set(float64(stats.HeapObjects))
	m.RuntimePauseTotalNs.Set(float64(stats.PauseTotalNs))
	m.RuntimeGC.Set(float64(stats.NumGC))
}

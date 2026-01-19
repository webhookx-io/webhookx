package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/services/schedule"
	"go.uber.org/zap"
)

type Metrics struct {
	Enabled  bool
	Interval time.Duration

	scheduler schedule.Scheduler

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

func (m *Metrics) Name() string {
	return "metrics"
}

func (m *Metrics) Stop(ctx context.Context) error {
	if m.Enabled {
		return StopOpentelemetry(ctx)
	}
	return nil
}

func New(cfg modules.MetricsConfig, scheduler schedule.Scheduler) (*Metrics, error) {
	m := &Metrics{
		Enabled:   len(cfg.Exports) > 0,
		Interval:  time.Second * time.Duration(cfg.PushInterval),
		scheduler: scheduler,
	}

	if m.Enabled {
		err := SetupOpentelemetry(cfg.Attributes, cfg.Opentelemetry, m)
		if err != nil {
			return nil, err
		}

		zap.S().Infof("enabled metric exports: %v", cfg.Exports)
	}

	return m, nil
}

func (m *Metrics) Start() error {
	if m.Enabled {
		m.scheduler.AddTask(&schedule.Task{
			Name:     "metrics.collectRuntimeStats",
			Interval: m.Interval,
			Do:       m.collectRuntimeStats,
		})
	}
	return nil
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

package metrics

import (
	"context"

	"github.com/go-kit/kit/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type LabelValues []string

func (lvs LabelValues) With(labelValues ...string) LabelValues {
	if len(labelValues)%2 != 0 {
		labelValues = append(labelValues, "unknown")
	}
	return append(lvs, labelValues...)
}

func (lvs LabelValues) ToLabels() []attribute.KeyValue {
	labels := make([]attribute.KeyValue, len(lvs)/2)
	for i := 0; i < len(labels); i++ {
		labels[i] = attribute.String(lvs[2*i], lvs[2*i+1])
	}
	return labels
}

type Counter struct {
	lvs LabelValues
	c   metric.Float64Counter
}

func (c *Counter) With(labelValues ...string) metrics.Counter {
	return &Counter{
		lvs: c.lvs.With(labelValues...),
		c:   c.c,
	}
}

func (c *Counter) Add(delta float64) {
	c.c.Add(context.Background(), delta, metric.WithAttributes(c.lvs.ToLabels()...))
}

func NewCounter(meter metric.Meter, name string, desc string) *Counter {
	c, _ := meter.Float64Counter(
		name,
		metric.WithDescription(desc),
		metric.WithUnit("1"),
	)
	return &Counter{
		c: c,
	}
}

type Gauge struct {
	lvs LabelValues
	g   metric.Float64Gauge
}

func NewGauge(meter metric.Meter, name string, desc string) *Gauge {
	g, _ := meter.Float64Gauge(
		name,
		metric.WithDescription(desc),
		metric.WithUnit("1"),
	)
	return &Gauge{
		g: g,
	}
}

func (g *Gauge) With(labelValues ...string) metrics.Gauge {
	return &Gauge{
		lvs: g.lvs.With(labelValues...),
		g:   g.g,
	}
}

func (g *Gauge) Add(delta float64) {
	g.g.Record(context.Background(), delta, metric.WithAttributes(g.lvs.ToLabels()...))
}

func (g *Gauge) Set(delta float64) {
	g.g.Record(context.Background(), delta, metric.WithAttributes(g.lvs.ToLabels()...))
}

type Histogram struct {
	lvs LabelValues
	h   metric.Float64Histogram
}

func NewHistogram(meter metric.Meter, name string, desc string, unit string) *Histogram {
	h, _ := meter.Float64Histogram(
		name,
		metric.WithDescription(desc),
		metric.WithUnit(unit),
		metric.WithExplicitBucketBoundaries(.005, .01, .025, .05, .075, .1, .25, .5, .75, 1, 2.5, 5, 7.5, 10),
	)
	return &Histogram{
		h: h,
	}
}

func (h *Histogram) With(labelValues ...string) metrics.Histogram {
	return &Histogram{
		lvs: h.lvs.With(labelValues...),
		h:   h.h,
	}
}

func (h *Histogram) Observe(value float64) {
	h.h.Record(context.Background(), value, metric.WithAttributes(h.lvs.ToLabels()...))
}

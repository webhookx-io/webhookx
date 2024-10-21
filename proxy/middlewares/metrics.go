package middlewares

import (
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"net/http"
	"time"
)

type MetricsMiddleware struct {
	metrics *metrics.Metrics
}

func NewMetricsMiddleware(metrics *metrics.Metrics) *MetricsMiddleware {
	return &MetricsMiddleware{
		metrics: metrics,
	}
}

func (m *MetricsMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.metrics.ProxyRequestCounter.Add(1)
		start := time.Now()
		next.ServeHTTP(w, r)
		m.metrics.ProxyRequestDurationHistogram.Observe(time.Since(start).Seconds())
	})
}

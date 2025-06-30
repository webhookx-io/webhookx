package status

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/pkg/http/middlewares"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/status/health"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net/http"
	"net/http/pprof"
	"runtime"
	"time"
)

type API struct {
	debugEndpoints bool
	tracer         *tracing.Tracer
	accessLogger   accesslog.AccessLogger
	indicators     []*health.Indicator
}

func (api *API) Status(w http.ResponseWriter, r *http.Request) {
	var mstats runtime.MemStats
	runtime.ReadMemStats(&mstats)

	data := stats.Collect()

	startedAt := data.Time("started_at")

	resp := StatusResponse{
		UpTime: time.Since(startedAt).Round(time.Second).String(),
		Runtime: RuntimeStats{
			Go:         runtime.Version(),
			Goroutines: runtime.NumGoroutine(),
		},
		Memory: MemoryStats{
			Alloc:       fmt.Sprintf("%.2f MiB", BytesToMiB(mstats.Alloc)),
			Sys:         fmt.Sprintf("%.2f MiB", BytesToMiB(mstats.Sys)),
			HeapAlloc:   fmt.Sprintf("%.2f MiB", BytesToMiB(mstats.HeapAlloc)),
			HeapIdle:    fmt.Sprintf("%.2f MiB", BytesToMiB(mstats.HeapIdle)),
			HeapInuse:   fmt.Sprintf("%.2f MiB", BytesToMiB(mstats.HeapInuse)),
			HeapObjects: int64(mstats.HeapObjects),
			GC:          int64(mstats.NumGC),
		},
		Database: DatabaseStats{
			TotalConnections:  data.Int("database.total_connections"),
			ActiveConnections: data.Int("database.active_connections"),
		},
		InboundRequests:            data.Int64("gateway.requests"),
		InboundFailedRequests:      data.Int64("gateway.failed_requests"),
		OutboundRequests:           data.Int64("outbound.requests"),
		OutboundProcessingRequests: data.Int64("outbound.processing_requests"),
		OutboundFailedRequests:     data.Int64("outbound.failed_requests"),
		Queue: QueueStats{
			Size:           data.Int64("queue.size"),
			BacklogLatency: data.Int64("queue.backlog_latency"),
		},
		Event: EventStats{
			Pending: data.Int64("eventqueue.size"),
		},
	}

	response.JSON(w, http.StatusOK, resp)
}

func (api *API) Health(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:     health.StatusUp,
		Components: make(map[string]HealthResult),
	}
	for _, check := range api.indicators {
		res := HealthResult{
			Status: health.StatusUp,
			Error:  nil,
		}
		err := check.Check()
		if err != nil {
			resp.Status = health.StatusDown

			res.Status = health.StatusDown
			res.Error = utils.Pointer(err.Error())
		}
		resp.Components[check.Name] = res
	}

	if resp.Status != health.StatusUp {
		response.JSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (api *API) Handler() http.Handler {
	r := mux.NewRouter()

	if api.accessLogger != nil {
		r.Use(accesslog.NewMiddleware(api.accessLogger))
	}

	if api.tracer != nil {
		r.Use(otelhttp.NewMiddleware("api.status"))
	}
	r.Use(middlewares.PanicRecovery)

	r.HandleFunc("/", api.Status).Methods("GET")
	r.HandleFunc("/health", api.Health).Methods("GET")

	if api.debugEndpoints {
		r.HandleFunc("/debug/pprof/profile", pprof.Profile).Methods("GET")
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol).Methods("GET")
		r.HandleFunc("/debug/pprof/trace", pprof.Trace).Methods("GET")
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline).Methods("GET")
		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index).Methods("GET")
	}

	return r
}

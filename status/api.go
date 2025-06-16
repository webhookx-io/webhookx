package status

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/pkg/http/middlewares"
	"github.com/webhookx-io/webhookx/pkg/http/response"
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
	startAt        time.Time
	debugEndpoints bool
	tracer         *tracing.Tracer
	accessLogger   accesslog.AccessLogger
	indicators     []*health.Indicator
}

func (api *API) Index(w http.ResponseWriter, r *http.Request) {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	resp := StatusResponse{
		UpTime: time.Since(api.startAt).Round(time.Second).String(),
		Memory: MemoryStatus{
			Alloc: fmt.Sprintf("%.2f MiB", BytesToMiB(stats.Alloc)),
			Sys:   fmt.Sprintf("%.2f MiB", BytesToMiB(stats.Sys)),
			GC:    int64(stats.NumGC),
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

	r.HandleFunc("/", api.Index).Methods("GET")
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

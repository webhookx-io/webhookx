package status

import (
	"context"
	"fmt"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/status/health"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

type Status struct {
	api *API
	cfg *config.StatusConfig
	s   *http.Server
	log *zap.SugaredLogger
}

type Options struct {
	AccessLog  accesslog.AccessLogger
	Config     *config.Config
	Indicators []*health.Indicator
}

func NewStatus(cfg config.StatusConfig, tracer *tracing.Tracer, opts Options) *Status {
	api := &API{
		debugEndpoints: cfg.DebugEndpoints,
		tracer:         tracer,
		accessLogger:   opts.AccessLog,
		indicators:     opts.Indicators,
	}
	s := &http.Server{
		Handler:      api.Handler(),
		Addr:         cfg.Listen,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	status := &Status{
		api: api,
		cfg: &cfg,
		s:   s,
		log: zap.S().Named("status"),
	}

	return status
}

func (a *Status) Start() {
	go func() {
		if err := a.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("Failed to start status HTTP server: %v", err)
			os.Exit(1)
		}
	}()

	a.log.Infow(fmt.Sprintf(`listening on address "%s"`, a.cfg.Listen))

	if a.cfg.DebugEndpoints {
		a.log.Infow("serving debug endpoints at /debug", "pprof", "/debug/pprof/")
	}
}

func (a *Status) Stop() error {
	if err := a.s.Shutdown(context.TODO()); err != nil {
		return err
	}
	a.log.Infof("status stopped")
	return nil
}

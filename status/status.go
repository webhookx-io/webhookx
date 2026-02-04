package status

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/status/health"
	"go.uber.org/zap"
)

var (
	TestIndicators []*health.Indicator // only for testing
)

type Status struct {
	api *API
	cfg *modules.StatusConfig
	s   *http.Server
	log *zap.SugaredLogger
}

type Options struct {
	AccessLog  accesslog.AccessLogger
	Indicators []*health.Indicator
}

func NewStatus(cfg modules.StatusConfig, opts Options) *Status {
	api := &API{
		debugEndpoints: cfg.DebugEndpoints,
		accessLogger:   opts.AccessLog,
		indicators:     append(opts.Indicators, TestIndicators...),
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

func (s *Status) Name() string {
	return "status"
}

func (s *Status) Start() error {
	go func() {
		if err := s.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("Failed to start status HTTP server: %v", err)
			os.Exit(1)
		}
	}()

	s.log.Infow(fmt.Sprintf(`listening on address "%s"`, s.cfg.Listen))

	if s.cfg.DebugEndpoints {
		s.log.Infow("serving debug endpoints at /debug", "pprof", "/debug/pprof/")
	}
	return nil
}

func (s *Status) Stop(ctx context.Context) error {
	s.log.Infof("exiting")
	if err := s.s.Shutdown(ctx); err != nil {
		return err
	}
	s.log.Infof("exit")
	return nil
}

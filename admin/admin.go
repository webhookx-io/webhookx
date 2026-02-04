package admin

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"go.uber.org/zap"
)

// Admin is an HTTP Server
type Admin struct {
	cfg *modules.AdminConfig
	s   *http.Server
	log *zap.SugaredLogger
}

func NewAdmin(cfg modules.AdminConfig, handler http.Handler) *Admin {
	s := &http.Server{
		Handler: handler,
		Addr:    cfg.Listen,

		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}

	admin := &Admin{
		cfg: &cfg,
		s:   s,
		log: zap.S().Named("admin"),
	}

	return admin
}

func (a *Admin) Name() string {
	return "admin"
}

// Start starts an HTTP server
func (a *Admin) Start() error {
	go func() {
		tls := a.cfg.TLS
		if tls.Enabled() {
			if err := a.s.ListenAndServeTLS(tls.Cert, tls.Key); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start admin HTTPS server: %v", err)
				os.Exit(1)
			}
		} else {
			if err := a.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start admin HTTP server: %v", err)
				os.Exit(1)
			}
		}
	}()

	a.log.Infow(fmt.Sprintf(`listening on address "%s"`, a.cfg.Listen),
		"tls", a.cfg.TLS.Enabled())

	if a.cfg.DebugEndpoints {
		a.log.Infow("serving debug endpoints at /debug", "pprof", "/debug/pprof/")
	}
	return nil
}

// Stop stops the HTTP server
func (a *Admin) Stop(ctx context.Context) error {
	a.log.Infof("exiting")
	if err := a.s.Shutdown(ctx); err != nil {
		return err
	}
	a.log.Infof("exit")
	return nil
}

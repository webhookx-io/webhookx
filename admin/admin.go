package admin

import (
	"context"
	"github.com/webhookx-io/webhookx/config"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

// Admin is an HTTP Server
type Admin struct {
	cfg *config.AdminConfig
	s   *http.Server
}

func NewAdmin(cfg config.AdminConfig, handler http.Handler) *Admin {
	s := &http.Server{
		Handler: handler,
		Addr:    cfg.Listen,

		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}

	admin := &Admin{
		cfg: &cfg,
		s:   s,
	}

	return admin
}

// Start starts an HTTP server
func (a *Admin) Start() {
	go func() {
		tls := a.cfg.TLS
		if tls.Enabled() {
			if err := a.s.ListenAndServeTLS(tls.Cert, tls.Key); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start Admin : %v", err)
				os.Exit(1)
			}
		} else {
			if err := a.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start Admin : %v", err)
				os.Exit(1)
			}
		}
	}()
}

// Stop stops the HTTP server
func (a *Admin) Stop() error {
	// TODO shutdown timeout
	if err := a.s.Shutdown(context.TODO()); err != nil {
		// Error from closing listeners, or context timeout:
		return err
	}
	return nil
}

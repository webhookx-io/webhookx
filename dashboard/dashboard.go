package dashboard

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/ui"
	"go.uber.org/zap"
)

type Dashboard struct {
	cfg modules.DashboardConfig
	log *zap.SugaredLogger
	s   *http.Server
}

func NewDashboard(cfg modules.DashboardConfig) (*Dashboard, error) {
	adminURL, err := url.Parse(cfg.AdminAddress)
	if err != nil {
		return nil, err
	}

	distFS, err := fs.Sub(ui.DistFiles, "dist")
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		Addr:         cfg.Listen,
		Handler:      NewAPI(adminURL, distFS).Handler(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return &Dashboard{
		cfg: cfg,
		s:   server,
		log: zap.S().Named("dashboard"),
	}, nil
}

func (d *Dashboard) Name() string {
	return "dashboard"
}

// Start starts the HTTP server
func (d *Dashboard) Start() error {
	go func() {
		tls := d.cfg.TLS
		if tls.Enabled() {
			if err := d.s.ListenAndServeTLS(tls.Cert, tls.Key); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start dashboard HTTPS server: %v", err)
				os.Exit(1)
			}
		} else {
			if err := d.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start dashboard HTTP server: %v", err)
				os.Exit(1)
			}
		}
	}()

	d.log.Infow(fmt.Sprintf(`listening on address "%s"`, d.cfg.Listen), "tls", d.cfg.TLS.Enabled())

	return nil
}

// Stop stops the HTTP server
func (d *Dashboard) Stop(ctx context.Context) error {
	d.log.Infof("exiting")
	if err := d.s.Shutdown(ctx); err != nil {
		return err
	}
	d.log.Infof("exit")
	return nil
}

package server

import (
	"context"
	"fmt"
	"github.com/webhookx-io/webhookx/internal/config"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server is an HTTP Server
type Server struct {
	s *http.Server

	stopChan chan bool
}

func NewServer(cfg config.ServerConfig, handler http.Handler) *Server {
	s := &http.Server{
		Handler: handler,
		Addr:    fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),

		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,

		// TODO: expose more to be configurable
	}

	srv := &Server{
		s:        s,
		stopChan: make(chan bool, 1),
	}

	return srv
}

// Start starts an HTTP server
func (s *Server) Start() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		zap.S().Infof("WebhookX Server is shutting down")
		s.Stop()
	}()

	go func() {
		if err := s.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("Failed to start Server : %v", err)
			os.Exit(1)
		}
	}()
}

// Stop stops the HTTP server
func (s *Server) Stop() {
	defer zap.S().Infof("WebhookX Server stopped")
	if err := s.s.Shutdown(context.TODO()); err != nil {
		// Error from closing listeners, or context timeout:
		zap.S().Errorf("WebhookX shutdown: %v", err)
	}
	s.stopChan <- true
}

func (s *Server) Wait() {
	<-s.stopChan
}

func (s *Server) Close() {
	close(s.stopChan)
}

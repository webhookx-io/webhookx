package webhookx

import (
	"github.com/webhookx-io/webhookx/internal/api"
	"github.com/webhookx-io/webhookx/internal/config"
	"github.com/webhookx-io/webhookx/internal/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Start(cfg *config.Config) error {
	initLogger(&cfg.Log)

	srv, err := initServer(cfg)
	if err != nil {
		return err
	}

	srv.Start()
	defer srv.Close()

	srv.Wait()
	return nil
}

func initLogger(cfg *config.LogConfig) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		panic(err)
	}
	log := zap.Must(zap.NewDevelopment(zap.AddStacktrace(zap.PanicLevel), zap.IncreaseLevel(level)))
	zap.ReplaceGlobals(log)
}

func initServer(cfg *config.Config) (*server.Server, error) {
	api, err := api.NewAPI(cfg)
	if err != nil {
		return nil, err
	}

	srv := server.NewServer(cfg.ServerConfig, api.Handler())
	return srv, nil
}

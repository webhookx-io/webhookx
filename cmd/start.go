package cmd

import (
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/server"
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

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Start(cfg); err != nil {
				return err
			}
			return nil
		},
	}
}

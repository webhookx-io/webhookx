package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfg *config.Config

	cmd = &cobra.Command{
		Use:          "webhookx",
		Short:        "",
		Long:         ``,
		SilenceUsage: true,
	}
)

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newMigrationsCmd())
	cmd.AddCommand(newStartCmd())
}

func initConfig() {
	var err error
	cfg, err = config.Init()
	cobra.CheckErr(err)
	fmt.Println("configuration:", cfg)
}

func initLogger() {
	level, err := zapcore.ParseLevel(cfg.Log.Level)
	cobra.CheckErr(err)
	log := zap.Must(zap.NewDevelopment(zap.AddStacktrace(zap.PanicLevel), zap.IncreaseLevel(level)))
	zap.ReplaceGlobals(log)
}

func Execute() {
	cmd.Execute()
}

package cmd

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
)

var (
	AdminURL = "http://localhost:9601"
)

var (
	configurationFile string
	verbose           bool
)

func initConfig(filename string) (*config.Config, error) {
	cfg := config.New()
	if err := config.Load(filename, cfg); err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}
	return cfg, nil
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "webhookx",
		Short:        "",
		Long:         ``,
		SilenceUsage: true,
	}

	cmd.SetOut(os.Stdout)
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "", false, "Verbose logging.")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newDatabaseCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newAdminCmd())

	return cmd
}

func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

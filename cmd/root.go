package cmd

import (
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
	"os"
)

const (
	defaultAdminURL = "http://localhost:8080"
)

var (
	configurationFile string
	verbose           bool
	cfg               *config.Config
)

func initConfig() {
	var err error
	if configurationFile != "" {
		cfg, err = config.InitWithFile(configurationFile)
	} else {
		cfg, err = config.Init()
	}
	cobra.CheckErr(err)

	err = cfg.Validate()
	cobra.CheckErr(err)
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "webhookx",
		Short:        "",
		Long:         ``,
		SilenceUsage: true,
	}
	cobra.OnInitialize(initConfig)

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

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
	"os"
)

var (
	AdminURL = "http://localhost:9601"
)

var (
	configurationFile string
	verbose           bool
	cfg               *config.Config
)

func initConfig() {
	var err error

	var options config.Options
	if configurationFile != "" {
		buf, err := os.ReadFile(configurationFile)
		cobra.CheckErr(err)
		options.YAML = buf
	}

	cfg, err = config.New(&options)
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

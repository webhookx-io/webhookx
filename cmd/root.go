package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
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
	cobra.OnInitialize(initConfig)

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

func Execute() {
	cmd.Execute()
}

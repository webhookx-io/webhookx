package cmd

import (
	"github.com/spf13/cobra"
)

var (
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
}

func initConfig() {

}

func Execute() {
	cmd.Execute()
}

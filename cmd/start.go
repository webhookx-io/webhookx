package cmd

import (
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/app"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := app.NewApp(cfg)
			if err != nil {
				return err
			}
			if err := app.Start(); err != nil {
				return err
			}
			return nil
		},
	}
}

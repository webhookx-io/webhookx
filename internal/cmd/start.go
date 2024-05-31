package cmd

import (
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/internal"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := webhookx.Start(cfg); err != nil {
				return err
			}
			return nil
		},
	}
}

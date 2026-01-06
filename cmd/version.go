package cmd

import (
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Long:  `Print the version with a short commit hash.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("WebhookX %s (%s)\n", webhookx.VERSION, webhookx.COMMIT)
		},
	}
}

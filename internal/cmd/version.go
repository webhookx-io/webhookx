package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/internal/config"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Long:  `The version command prints the version of WebhookX along with a short commit hash.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("WebhookX %s (%s) \n", config.VERSION, config.COMMIT)
		},
	}
}

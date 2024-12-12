package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"time"
)

func newAdminDumpCmd() *cobra.Command {
	var (
		addr      string
		timeout   int
		workspace string
	)

	dump := &cobra.Command{
		Use:   "dump",
		Short: "Dump entities to declarative configuration",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/workspaces/%s/config/dump", addr, workspace)
			r, err := http.NewRequest("POST", url, nil)
			if err != nil {
				return err
			}

			content, err := sendHTTPRequest(r, time.Duration(timeout)*time.Second)
			if err != nil {
				return err
			}

			cmd.Print(content)
			return nil
		},
	}

	dump.Flags().StringVarP(&workspace, "workspace", "", "default", "Set a specific workspace.")
	dump.Flags().StringVarP(&addr, "addr", "", defaultAdminURL, "HTTP address of WebhookX's Admin API.")
	dump.Flags().IntVarP(&timeout, "timeout", "", 10, "Set the request timeout for the client to connect with WebhookX (in seconds).")

	return dump
}

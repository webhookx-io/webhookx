package cmd

import (
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"time"
)

func newAdminSyncCmd() *cobra.Command {
	var (
		addr      string
		timeout   int
		workspace string
	)

	sync := &cobra.Command{
		Use:   "sync [flags] filename",
		Short: "Synchronize a declarative configuration to WebhookX.",
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]
			b, err := os.ReadFile(filename)
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/workspaces/%s/config/sync", addr, workspace)
			r, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
			if err != nil {
				return err
			}

			r.Header.Add("Content-Type", "text/plain")
			_, err = sendHTTPRequest(r, time.Duration(timeout)*time.Second)
			if err != nil {
				return err
			}

			return nil
		},
	}

	sync.Flags().StringVarP(&workspace, "workspace", "", "default", "Set a specific workspace.")
	sync.Flags().StringVarP(&addr, "addr", "", defaultAdminURL, "HTTP address of WebhookX's Admin API.")
	sync.Flags().IntVarP(&timeout, "timeout", "", 10, "Set the request timeout for the client to connect with WebhookX (in seconds).")

	return sync
}

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
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
			r.Header.Set("User-Agent", "WebhookX/"+config.VERSION)
			r.Header.Set("Content-Type", "text/plain")

			if verbose {
				requestDump, err := httputil.DumpRequestOut(r, true)
				if err != nil {
					return err
				}
				cmd.Println(string(requestDump))
			}

			client := http.Client{
				Timeout: time.Duration(timeout) * time.Second,
			}
			resp, err := client.Do(r)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if verbose {
				responseDump, err := httputil.DumpResponse(resp, true)
				if err != nil {
					return err
				}
				cmd.Println(string(responseDump))
			}

			b, err = io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("invalid status code: %d %s", resp.StatusCode, string(b))
			}

			cmd.Println("sync successfully")

			return nil
		},
	}

	sync.Flags().StringVarP(&workspace, "workspace", "", "default", "Set a specific workspace.")
	sync.Flags().StringVarP(&addr, "addr", "", AdminURL, "HTTP address of WebhookX's Admin API.")
	sync.Flags().IntVarP(&timeout, "timeout", "", 10, "Set the request timeout for the client to connect with WebhookX (in seconds).")

	return sync
}

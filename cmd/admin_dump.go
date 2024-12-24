package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/config"
	"io"
	"net/http"
	"net/http/httputil"
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
			r.Header.Set("User-Agent", "WebhookX/"+config.VERSION)

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

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("invalid status code: %d %s", resp.StatusCode, string(b))
			}

			cmd.Print(string(b))

			return nil
		},
	}

	dump.Flags().StringVarP(&workspace, "workspace", "", "default", "Set a specific workspace.")
	dump.Flags().StringVarP(&addr, "addr", "", defaultAdminURL, "HTTP address of WebhookX's Admin API.")
	dump.Flags().IntVarP(&timeout, "timeout", "", 10, "Set the request timeout for the client to connect with WebhookX (in seconds).")

	return dump
}

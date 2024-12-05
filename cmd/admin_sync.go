package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io"
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

			url := fmt.Sprintf("%s/workspaces/%s/sync", addr, workspace)

			return send(url, b, time.Duration(timeout)*time.Second)
		},
	}

	sync.Flags().StringVarP(&workspace, "workspace", "", "default", "Set a specific workspace.")
	sync.Flags().StringVarP(&addr, "addr", "", defaultAdminURL, "HTTP address of WebhookX's Admin API.")
	sync.Flags().IntVarP(&timeout, "timeout", "", 10, "Set the request timeout for the client to connect with WebhookX (in seconds).")

	return sync
}

func send(url string, data []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	r.Header.Add("Content-Type", "text/plain")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("timeout")
		}
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

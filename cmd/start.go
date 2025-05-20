package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/app"
	"os"
	"os/signal"
	"syscall"
)

func newStartCmd() *cobra.Command {
	start := &cobra.Command{
		Use:   "start",
		Short: "Start server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := app.New(cfg)
			if err != nil {
				return err
			}

			ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-ctx.Done()
				err = app.Stop()
				if err != nil {
					os.Exit(1)
				}
			}()

			if err := app.Start(); err != nil {
				return err
			}

			app.Wait()

			return nil
		},
	}

	start.PersistentFlags().StringVarP(&configurationFile, "config", "", "", "The configuration filename")

	return start
}

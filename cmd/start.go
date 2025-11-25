package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/pkg/license"
)

func newStartCmd() *cobra.Command {
	start := &cobra.Command{
		Use:   "start",
		Short: "Start server",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			lic, err := license.Load()
			if err != nil {
				return err
			}
			license.SetLicenser(license.NewLicenser(lic))

			cfg, err := initConfig(configurationFile)
			if err != nil {
				return err
			}

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

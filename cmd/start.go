package cmd

import (
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
			return app.Run()
		},
	}

	start.PersistentFlags().StringVarP(&configurationFile, "config", "", "", "The configuration filename")

	return start
}

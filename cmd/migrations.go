package cmd

import (
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/db/migrator"
)

func newMigrationsResetCmd() *cobra.Command {
	var yes bool
	reset := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if !prompt("Are you sure? This operation is irreversible.") {
					return errors.New("canceled")
				}
			}
			m := migrator.New(&cfg.DatabaseConfig)
			fmt.Println("resetting database...")
			if err := m.Reset(); err != nil {
				return err
			}
			fmt.Println("database successfully reset")
			return nil
		},
	}
	reset.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "yes")
	return reset
}

func newMigrationsCmd() *cobra.Command {

	migration := &cobra.Command{
		Use:   "migrations",
		Short: "",
		Long:  ``,
	}

	migration.PersistentFlags().StringVarP(&configurationFile, "config", "", "", "The configuration filename")

	migration.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Print the migration status",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := migrator.New(&cfg.DatabaseConfig)
			version, dirty, err := m.Status()
			if err != nil {
				return err
			}

			if dirty {
				fmt.Printf("%d (dirty)\n", version)
			} else {
				fmt.Printf("%d\n", version)
			}
			return nil
		},
	})

	migration.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run any new migrations",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := migrator.New(&cfg.DatabaseConfig)
			if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return err
			}
			fmt.Println("database is up-to-date")
			return nil
		},
	})

	migration.AddCommand(newMigrationsResetCmd())

	return migration
}

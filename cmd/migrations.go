package cmd

import (
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/db/migrator"
)

func newMigrationsCmd() *cobra.Command {

	migration := &cobra.Command{
		Use:   "migrations",
		Short: "",
		Long:  ``,
	}

	var (
		timeout int
	)
	migration.PersistentFlags().IntVarP(&timeout, "timeout", "", 0, "timeout seconds")
	migration.PersistentFlags().IntVarP(&timeout, "lock-timeout", "", 0, "lock timeout seconds")

	migration.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "print the migration status",
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
		Short: "run any new migrations",
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

	migration.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "reset the database",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			// fixme: add promp
			m := migrator.New(&cfg.DatabaseConfig)
			fmt.Println("resetting database...")
			if err := m.Reset(); err != nil {
				return err
			}
			fmt.Println("database successfully reset")
			return nil
		},
	})

	return migration
}

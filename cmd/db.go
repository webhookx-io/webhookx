package cmd

import (
	"errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/spf13/cobra"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/migrator"
)

var (
	quiet bool
)

func newDatabaseResetCmd() *cobra.Command {
	var yes bool
	reset := &cobra.Command{
		Use:   "reset",
		Short: "Reset the database",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if !prompt(cmd.OutOrStdout(), "Are you sure? This operation is irreversible.") {
					return errors.New("canceled")
				}
			}
			db, err := db.NewSqlDB(cfg.Database)
			if err != nil {
				return err
			}
			m := migrator.New(db, &migrator.Options{Quiet: quiet})
			if !quiet {
				cmd.Println("resetting database...")
			}
			if err := m.Reset(); err != nil {
				return err
			}
			if !quiet {
				cmd.Println("database successfully reset")
			}
			return nil
		},
	}
	reset.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "yes")
	return reset
}

func newDatabaseCmd() *cobra.Command {

	database := &cobra.Command{
		Use:   "db",
		Short: "Database commands",
		Long:  ``,
	}

	database.PersistentFlags().StringVarP(&configurationFile, "config", "", "", "The configuration filename")
	database.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")

	database.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Print the migration status",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := db.NewSqlDB(cfg.Database)
			if err != nil {
				return err
			}
			m := migrator.New(db, &migrator.Options{Quiet: quiet})
			status, err := m.Status()
			if err != nil {
				return err
			}
			cmd.Println(status)
			return nil
		},
	})

	database.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Run any new migrations",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := db.NewSqlDB(cfg.Database)
			if err != nil {
				return err
			}
			m := migrator.New(db, &migrator.Options{Quiet: quiet})
			if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return err
			}
			cmd.Println("database is up-to-date")
			return nil
		},
	})

	database.AddCommand(newDatabaseResetCmd())

	return database
}

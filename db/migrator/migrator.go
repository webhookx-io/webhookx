package migrator

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/migrations"
)

// Migrator is a database migrator
type Migrator struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Migrator {
	return &Migrator{
		cfg: cfg,
	}
}

func (m *Migrator) init() (*migrate.Migrate, error) {
	db, err := m.cfg.DatabaseConfig.GetDB()
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: m.cfg.DatabaseConfig.Database,
	})
	if err != nil {
		return nil, err
	}

	d, err := iofs.New(migrations.SQLs, ".")
	if err != nil {
		return nil, err
	}

	return migrate.NewWithInstance("iofs", d, "postgres", driver)
}

func (m *Migrator) Bootstrap() error {
	panic("implement me")
}

// Reset reset database
func (m *Migrator) Reset() error {
	migrate, err := m.init()
	if err != nil {
		return err
	}
	return migrate.Drop()
}

func (m *Migrator) Up() error {
	migrate, err := m.init()
	if err != nil {
		return err
	}
	return migrate.Up()
}

func (m *Migrator) Down() error {
	migrate, err := m.init()
	if err != nil {
		return err
	}
	return migrate.Down()
}

// Status returns the current status
func (m *Migrator) Status() (version uint, dirty bool, err error) {
	migrate, err := m.init()
	if err != nil {
		return 0, false, err
	}
	return migrate.Version()
}

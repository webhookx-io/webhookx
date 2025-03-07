package migrator

import (
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/migrations"
	"github.com/webhookx-io/webhookx/utils"
)

type Migrator struct {
	cfg *config.DatabaseConfig
}

func New(cfg *config.DatabaseConfig) *Migrator {
	migrator := &Migrator{
		cfg: cfg,
	}
	return migrator
}

func (m *Migrator) init() (*migrate.Migrate, error) {
	db, err := db.NewSqlDB(*m.cfg)
	if err != nil {
		return nil, err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: m.cfg.Database,
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
	err = migrate.Up()
	if err != nil {
		return err
	}

	return m.initDefaultWorkspace()
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

func (m *Migrator) initDefaultWorkspace() error {
	db, err := db.NewSqlDB(*m.cfg)
	if err != nil {
		return err
	}
	sql := `INSERT INTO workspaces(id, name) VALUES($1, 'default') ON CONFLICT(name) DO NOTHING;`
	_, err = db.Exec(sql, utils.KSUID())
	return err
}

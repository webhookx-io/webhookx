package migrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/webhookx-io/webhookx/db/migrations"
	"github.com/webhookx-io/webhookx/utils"
)

type Log struct{}

func (Log) Printf(format string, v ...interface{}) { fmt.Printf(format, v...) }

func (Log) Verbose() bool { return false }

type Migration struct {
	Version    int
	Identifier string
}

type Status struct {
	Version   int
	Dirty     bool
	Executeds []Migration
	Pendings  []Migration
}

func (s Status) String() string {
	var sb strings.Builder

	for _, m := range s.Executeds {
		if m.Version == s.Version && s.Dirty {
			sb.WriteString(fmt.Sprintf("%d %s (❌ dirty)\n", m.Version, m.Identifier))
		} else {
			sb.WriteString(fmt.Sprintf("%d %s (✅ executed)\n", m.Version, m.Identifier))
		}
	}

	if len(s.Pendings) > 0 {
		for _, m := range s.Pendings {
			sb.WriteString(fmt.Sprintf("%d %s (⏳ pending)\n", m.Version, m.Identifier))
		}
	}

	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Current version: %d\n", s.Version))
	sb.WriteString(fmt.Sprintf("  Dirty: %t\n", s.Dirty))
	sb.WriteString(fmt.Sprintf("  Executed: %d\n", len(s.Executeds)))
	sb.WriteString(fmt.Sprintf("  Pending: %d", len(s.Pendings)))

	return sb.String()

}

type Options struct {
	Quiet bool
}

type Migrator struct {
	migrations []Migration
	db         *sql.DB
	options    *Options
}

func New(db *sql.DB, opts *Options) *Migrator {
	if opts == nil {
		opts = &Options{}
	}
	migrator := &Migrator{
		db:      db,
		options: opts,
	}
	return migrator
}

func (m *Migrator) client() (*migrate.Migrate, error) {
	ctx := context.Background()
	conn, err := m.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	driver, err := postgres.WithConnection(ctx, conn, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	d, err := iofs.New(migrations.SQLs, ".")
	if err != nil {
		return nil, err
	}

	v, err := d.First()
	for {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return nil, err
		}
		_, identifier, e := d.ReadUp(v)
		if e != nil {
			return nil, e
		}
		m.migrations = append(m.migrations, Migration{
			Version:    int(v),
			Identifier: identifier,
		})
		v, err = d.Next(v)
	}

	client, err := migrate.NewWithInstance("iofs", d, "postgres", driver)
	if err != nil {
		return nil, err
	}
	if !m.options.Quiet {
		client.Log = &Log{}
	}
	client.LockTimeout = time.Second * 5

	return client, nil
}

// Reset reset database
func (m *Migrator) Reset() error {
	client, err := m.client()
	if err != nil {
		return err
	}
	defer func() { _, _ = client.Close() }()
	return client.Drop()
}

// Up runs db migrations
func (m *Migrator) Up() error {
	client, err := m.client()
	if err != nil {
		return err
	}
	defer func() { _, _ = client.Close() }()
	err = client.Up()
	if err != nil {
		return err
	}

	return m.initDefaultWorkspace()
}

// Status returns the current status
func (m *Migrator) Status() (status Status, err error) {
	client, err := m.client()
	if err != nil {
		return
	}
	defer func() { _, _ = client.Close() }()
	v, dirty, err := client.Version()
	if err != nil {
		switch {
		case errors.Is(err, migrate.ErrNilVersion):
			err = nil
		default:
		}
	}
	status.Version = int(v)
	status.Dirty = dirty

	for _, m := range m.migrations {
		if m.Version <= status.Version {
			status.Executeds = append(status.Executeds, m)
		} else {
			status.Pendings = append(status.Pendings, m)
		}
	}

	return
}

func (m *Migrator) initDefaultWorkspace() error {
	sql := `INSERT INTO workspaces(id, name) VALUES($1, 'default') ON CONFLICT(name) DO NOTHING;`
	_, err := m.db.Exec(sql, utils.KSUID())
	return err
}

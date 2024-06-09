package db

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/webhookx-io/webhookx/internal/config"
)

type DB struct {
	DB *sqlx.DB
}

func initSqlxDB(cfg config.PostgresConfig) (*sqlx.DB, error) {
	db, err := cfg.GetDB()
	if err != nil {
		return nil, err
	}
	return sqlx.NewDb(db, "postgres"), nil
}

func NewDB(cfg *config.Config) (*DB, error) {
	sqlxDB, err := initSqlxDB(cfg.PostgresConfig)
	if err != nil {
		return nil, err
	}

	db := &DB{
		DB: sqlxDB,
	}

	return db, nil
}

func (db *DB) Ping() error {
	return db.DB.Ping()
}

// TODO
func (db *DB) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := db.DB.Beginx()
	if err != nil {
		return err
	}

	// todo panic handler

	err = fn(ctx)

	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Wrap(err, rbErr.Error())
		}
		return err
	}

	return tx.Commit()
}

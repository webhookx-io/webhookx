package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/services/eventbus"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
)

type DB struct {
	DB  *sqlx.DB
	log *zap.SugaredLogger

	Workspaces       dao.WorkspaceDAO
	Endpoints        dao.EndpointDAO
	EndpointsWS      dao.EndpointDAO
	Events           dao.EventDAO
	EventsWS         dao.EventDAO
	Attempts         dao.AttemptDAO
	AttemptsWS       dao.AttemptDAO
	Sources          dao.SourceDAO
	SourcesWS        dao.SourceDAO
	AttemptDetails   dao.AttemptDetailDAO
	AttemptDetailsWS dao.AttemptDetailDAO
	Plugins          dao.PluginDAO
	PluginsWS        dao.PluginDAO
}

func NewSqlDB(cfg modules.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.GetDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(int(cfg.MaxPoolSize))
	db.SetMaxIdleConns(int(cfg.MaxPoolSize))
	db.SetConnMaxLifetime(time.Second * time.Duration(cfg.MaxLifetime))
	return db, nil
}

func NewDB(sqlDB *sql.DB, log *zap.SugaredLogger, bus eventbus.EventBus) (*DB, error) {
	sqlxDB := sqlx.NewDb(sqlDB, "pgx")

	opts := make([]dao.OptionFunc, 0)
	opts = append(opts, dao.WithPropagateHandler(func(ctx context.Context, opts *dao.Options, id string, entity interface{}) {
		data := &eventbus.CrudData{
			ID:        id,
			CacheName: opts.CacheName,
			Entity:    opts.EntityName,
			Data:      utils.Must(json.Marshal(entity)),
		}
		v := reflect.ValueOf(entity)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		wid := v.FieldByName("WorkspaceId")
		if wid.IsValid() {
			data.WID = wid.String()
		}
		_ = bus.ClusteringBroadcast(ctx, eventbus.EventCRUD, data)
	}))
	if tracing.Enabled("dao") {
		opts = append(opts, dao.WithInstrumented())
	}

	db := &DB{
		DB:               sqlxDB,
		log:              log,
		Workspaces:       dao.NewWorkspaceDAO(sqlxDB, opts...),
		Endpoints:        dao.NewEndpointDAO(sqlxDB, opts...),
		EndpointsWS:      dao.NewEndpointDAO(sqlxDB, append(opts, dao.WithWorkspace(true))...),
		Events:           dao.NewEventDao(sqlxDB, opts...),
		EventsWS:         dao.NewEventDao(sqlxDB, append(opts, dao.WithWorkspace(true))...),
		Attempts:         dao.NewAttemptDao(sqlxDB, opts...),
		AttemptsWS:       dao.NewAttemptDao(sqlxDB, append(opts, dao.WithWorkspace(true))...),
		Sources:          dao.NewSourceDAO(sqlxDB, opts...),
		SourcesWS:        dao.NewSourceDAO(sqlxDB, append(opts, dao.WithWorkspace(true))...),
		AttemptDetails:   dao.NewAttemptDetailDao(sqlxDB, opts...),
		AttemptDetailsWS: dao.NewAttemptDetailDao(sqlxDB, append(opts, dao.WithWorkspace(true))...),
		Plugins:          dao.NewPluginDAO(sqlxDB, opts...),
		PluginsWS:        dao.NewPluginDAO(sqlxDB, append(opts, dao.WithWorkspace(true))...),
	}

	return db, nil
}

func (db *DB) Ping() error {
	return db.DB.Ping()
}

func (db *DB) Stats() map[string]interface{} {
	stats := db.DB.Stats()
	return map[string]interface{}{
		"database.total_connections":  stats.OpenConnections,
		"database.active_connections": stats.InUse,
	}
}

func (db *DB) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := tracing.Start(ctx, "db.transaction")
	defer span.End()

	tx, err := db.DB.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			db.log.Errorf("panic recovered: %v", err)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Errorf("failed to rollback the tx: %v", rbErr)
			}
			panic(err)
		}
	}()

	ctx = transaction.WithTx(ctx, tx)

	err = fn(ctx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Wrap(err, rbErr.Error())
		}
		return err
	}

	return tx.Commit()
}

func (db *DB) Truncate(table string) error {
	sql := fmt.Sprintf("DELETE FROM %s", table)
	_, err := db.DB.Exec(sql)
	return err
}

func (db *DB) SqlDB() *sql.DB {
	return db.DB.DB
}

func (db *DB) Close() error {
	return db.DB.Close()
}

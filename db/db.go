package db

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
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

func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	sqlDB, err := cfg.InitSqlDB()
	if err != nil {
		return nil, err
	}
	sqlxDB := sqlx.NewDb(sqlDB, "postgres")

	db := &DB{
		DB:               sqlxDB,
		log:              zap.S(),
		Workspaces:       dao.NewWorkspaceDAO(sqlxDB),
		Endpoints:        dao.NewEndpointDAO(sqlxDB, false),
		EndpointsWS:      dao.NewEndpointDAO(sqlxDB, true),
		Events:           dao.NewEventDao(sqlxDB, false),
		EventsWS:         dao.NewEventDao(sqlxDB, true),
		Attempts:         dao.NewAttemptDao(sqlxDB, false),
		AttemptsWS:       dao.NewAttemptDao(sqlxDB, true),
		Sources:          dao.NewSourceDAO(sqlxDB, false),
		SourcesWS:        dao.NewSourceDAO(sqlxDB, true),
		AttemptDetails:   dao.NewAttemptDetailDao(sqlxDB, false),
		AttemptDetailsWS: dao.NewAttemptDetailDao(sqlxDB, true),
		Plugins:          dao.NewPluginDAO(sqlxDB, false),
		PluginsWS:        dao.NewPluginDAO(sqlxDB, true),
	}

	return db, nil
}

func (db *DB) Ping() error {
	return db.DB.Ping()
}

func (db *DB) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	tracingCtx, span := tracing.Start(ctx, "db.transaction", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	ctx = tracingCtx

	tx, err := db.DB.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			db.log.Errorf("[db] panic recovered: %v", err)
			if rbErr := tx.Rollback(); rbErr != nil {
				db.log.Errorf("[db] failed to rollback the tx: %v", rbErr)
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

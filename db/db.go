package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/transaction"
	"github.com/webhookx-io/webhookx/eventbus"
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

func NewDB(sqlDB *sql.DB, log *zap.SugaredLogger, bus *eventbus.EventBus) (*DB, error) {
	sqlxDB := sqlx.NewDb(sqlDB, "postgres")

	db := &DB{
		DB:               sqlxDB,
		log:              log,
		Workspaces:       dao.NewWorkspaceDAO(sqlxDB, bus),
		Endpoints:        dao.NewEndpointDAO(sqlxDB, bus, false),
		EndpointsWS:      dao.NewEndpointDAO(sqlxDB, bus, true),
		Events:           dao.NewEventDao(sqlxDB, bus, false),
		EventsWS:         dao.NewEventDao(sqlxDB, bus, true),
		Attempts:         dao.NewAttemptDao(sqlxDB, bus, false),
		AttemptsWS:       dao.NewAttemptDao(sqlxDB, bus, true),
		Sources:          dao.NewSourceDAO(sqlxDB, bus, false),
		SourcesWS:        dao.NewSourceDAO(sqlxDB, bus, true),
		AttemptDetails:   dao.NewAttemptDetailDao(sqlxDB, bus, false),
		AttemptDetailsWS: dao.NewAttemptDetailDao(sqlxDB, bus, true),
		Plugins:          dao.NewPluginDAO(sqlxDB, bus, false),
		PluginsWS:        dao.NewPluginDAO(sqlxDB, bus, true),
	}

	return db, nil
}

func (db *DB) Ping() error {
	return db.DB.Ping()
}

func (db *DB) TX(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := tracing.Start(ctx, "db.transaction", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

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

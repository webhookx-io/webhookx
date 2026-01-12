package dao

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
)

type eventDao struct {
	*DAO[entities.Event]
}

func NewEventDao(db *sqlx.DB, bus *eventbus.EventBus, workspace bool) EventDAO {
	opts := Options{
		Table:          "events",
		EntityName:     "event",
		Workspace:      workspace,
		CachePropagate: false,
		CacheName:      constants.EventCacheKey.Name,
	}
	return &eventDao{
		DAO: NewDAO[entities.Event](db, bus, opts),
	}
}

func (dao *eventDao) ListExistingUniqueIDs(ctx context.Context, uniques []string) (list []string, err error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.list_unique_ids", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	statement, args := psql.Select("unique_id").
		From(dao.opts.Table).
		Where(sq.Eq{"unique_id": uniques}).
		MustSql()
	dao.debugSQL(statement, args)
	err = dao.UnsafeDB(ctx).SelectContext(ctx, &list, statement, args...)
	return
}

func (dao *eventDao) BatchInsertIgnoreConflict(ctx context.Context, events []*entities.Event) (inserteds []string, err error) {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.batch_insert_ignore_conflict", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(events) == 0 {
		return
	}

	builder := psql.Insert(dao.opts.Table).Columns("id", "data", "event_type", "ingested_at", "ws_id", "unique_id")
	for _, event := range events {
		builder = builder.Values(event.ID, event.Data, event.EventType, event.IngestedAt, event.WorkspaceId, event.UniqueId)
	}
	statement, args := builder.Suffix("ON CONFLICT(id) DO NOTHING RETURNING id").MustSql()
	var rows *sqlx.Rows
	rows, err = dao.DB(ctx).QueryxContext(ctx, statement, args...)
	if err != nil {
		return
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return
		}
		inserteds = append(inserteds, id)
	}
	return inserteds, rows.Err()
}

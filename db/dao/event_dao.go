package dao

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/eventbus"
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
		CacheKey:       constants.EventCacheKey,
	}
	return &eventDao{
		DAO: NewDAO[entities.Event](db, bus, opts),
	}
}

func (dao *eventDao) BatchInsertIgnoreConflict(ctx context.Context, events []*entities.Event) (inserteds []string, err error) {
	if len(events) == 0 {
		return
	}

	builder := psql.Insert(dao.opts.Table).Columns("id", "data", "event_type", "ingested_at", "ws_id")
	for _, event := range events {
		builder = builder.Values(event.ID, event.Data, event.EventType, event.IngestedAt, event.WorkspaceId)
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

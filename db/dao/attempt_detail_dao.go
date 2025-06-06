package dao

import (
	"context"
	"fmt"
	"github.com/webhookx-io/webhookx/eventbus"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
)

type attemptDetailDao struct {
	*DAO[entities.AttemptDetail]
}

func NewAttemptDetailDao(db *sqlx.DB, bus *eventbus.EventBus, workspace bool) AttemptDetailDAO {
	opts := Options{
		Table:          "attempt_details",
		EntityName:     "attempt_detail",
		Workspace:      workspace,
		CachePropagate: false,
		CacheKey:       constants.AttemptDetailCacheKey,
	}
	return &attemptDetailDao{
		DAO: NewDAO[entities.AttemptDetail](db, bus, opts),
	}
}

func (dao *attemptDetailDao) BatchInsert(ctx context.Context, entities []*entities.AttemptDetail) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.batch_insert", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	now := time.Now()

	builder := psql.Insert(dao.opts.Table).Columns("id", "request_headers", "request_body", "response_headers", "response_body", "created_at", "updated_at", "ws_id")
	for _, entity := range entities {
		builder = builder.Values(entity.ID, entity.RequestHeaders, entity.RequestBody, entity.ResponseHeaders, entity.ResponseBody, now, now, entity.WorkspaceId)
	}
	sql, args := builder.Suffix(`
		ON CONFLICT (id) DO UPDATE SET
		request_headers = EXCLUDED.request_headers,
		request_body = EXCLUDED.request_body,
		response_headers = EXCLUDED.response_headers,
		response_body = EXCLUDED.response_body,
		updated_at = EXCLUDED.updated_at`).MustSql()
	dao.debugSQL(sql, args)
	result, err := dao.DB(ctx).ExecContext(ctx, sql, args...)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}

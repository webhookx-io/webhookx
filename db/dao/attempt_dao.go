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
	"github.com/webhookx-io/webhookx/pkg/types"
	"go.opentelemetry.io/otel/trace"
)

type attemptDao struct {
	*DAO[entities.Attempt]
}

type AttemptResult struct {
	Request     *entities.AttemptRequest
	Response    *entities.AttemptResponse
	AttemptedAt types.Time
	Status      entities.AttemptStatus
	ErrorCode   *entities.AttemptErrorCode
	Exhausted   bool
}

func NewAttemptDao(db *sqlx.DB, bus *eventbus.EventBus, workspace bool) AttemptDAO {
	opts := Options{
		Table:          "attempts",
		EntityName:     "attempt",
		Workspace:      workspace,
		CachePropagate: false,
		CacheKey:       constants.AttemptCacheKey,
	}
	return &attemptDao{
		DAO: NewDAO[entities.Attempt](db, bus, opts),
	}
}

func (dao *attemptDao) UpdateDelivery(ctx context.Context, id string, result *AttemptResult) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"request":      result.Request,
		"response":     result.Response,
		"attempted_at": result.AttemptedAt,
		"status":       result.Status,
		"error_code":   result.ErrorCode,
		"exhausted":    result.Exhausted,
		"updated_at":   sq.Expr("NOW()"),
	})
	return err
}

func (dao *attemptDao) UpdateStatusToQueued(ctx context.Context, ids []string) error {
	sql, args := psql.Update("attempts").
		Set("status", entities.AttemptStatusQueued).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     ids,
			"status": entities.AttemptStatusInit,
		}).MustSql()
	dao.debugSQL(sql, args)
	_, err := dao.DB(ctx).ExecContext(ctx, sql, args...)
	return err
}

func (dao *attemptDao) UpdateErrorCode(ctx context.Context, id string, status entities.AttemptStatus, code entities.AttemptErrorCode) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.update_error_code", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	_, err := dao.update(ctx, id, map[string]interface{}{
		"status":     status,
		"error_code": code,
		"updated_at": sq.Expr("NOW()"),
	})
	return err
}

func (dao *attemptDao) ListUnqueuedForUpdate(ctx context.Context, limit int) (list []*entities.Attempt, err error) {
	sql := "SELECT * FROM attempts WHERE status = 'INIT' and created_at <= now() AT TIME ZONE 'UTC' - INTERVAL '60 SECOND' limit $1 FOR UPDATE SKIP LOCKED"
	err = dao.UnsafeDB(ctx).SelectContext(ctx, &list, sql, limit)
	return
}

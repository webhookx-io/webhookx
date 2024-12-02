package dao

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	sq "github.com/Masterminds/squirrel"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
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

func NewAttemptDao(db *sqlx.DB, workspace bool) AttemptDAO {
	opts := Options{
		Table:          "attempts",
		EntityName:     "Attempt",
		Workspace:      workspace,
		CachePropagate: false,
		CacheKey:       constants.AttemptCacheKey,
	}
	return &attemptDao{
		DAO: NewDAO[entities.Attempt](db, opts),
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
	})
	return err
}

func (dao *attemptDao) UpdateStatus(ctx context.Context, id string, status entities.AttemptStatus) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"status": status,
	})
	return err
}

func (dao *attemptDao) UpdateStatusBatch(ctx context.Context, status entities.AttemptStatus, ids []string) error {
	sql, args := psql.Update("attempts").Set("status", status).Where(sq.Eq{"id": ids}).MustSql()
	_, err := dao.DB(ctx).ExecContext(ctx, sql, args...)
	return err
}

func (dao *attemptDao) UpdateErrorCode(ctx context.Context, id string, status entities.AttemptStatus, code entities.AttemptErrorCode) error {
	if tracer := tracing.TracerFromContext(ctx); tracer != nil {
		tracingCtx, span := tracer.Start(ctx, fmt.Sprintf("dao.%s.updateErrorCode", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		ctx = tracingCtx
	}
	_, err := dao.update(ctx, id, map[string]interface{}{
		"status":     status,
		"error_code": code,
	})
	return err
}

func (dao *attemptDao) ListUnqueued(ctx context.Context, limit int) (list []*entities.Attempt, err error) {
	sql := "SELECT * FROM attempts WHERE status = 'INIT' and created_at <= now() AT TIME ZONE 'UTC' - INTERVAL '60 SECOND' limit $1"
	err = dao.UnsafeDB(ctx).SelectContext(ctx, &list, sql, limit)
	return
}

package dao

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/types"
)

type attemptDao struct {
	*DAO[entities.Attempt]
}

type AttemptResult struct {
	ID          string
	Request     *entities.AttemptRequest
	Response    *entities.AttemptResponse
	AttemptedAt types.Time
	Status      entities.AttemptStatus
	ErrorCode   *entities.AttemptErrorCode
	Exhausted   bool
}

func NewAttemptDao(db *sqlx.DB, fns ...OptionFunc) AttemptDAO {
	opts := Options{
		Table:          "attempts",
		EntityName:     "attempt",
		CachePropagate: false,
		CacheName:      constants.AttemptCacheKey.Name,
	}
	return &attemptDao{DAO: NewDAO[entities.Attempt](db, opts, fns...)}
}

func (dao *attemptDao) UpdateDelivery(ctx context.Context, result *AttemptResult) error {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.update_result", dao.opts.Table))
	defer span.End()

	_, err := dao.update(ctx, result.ID, map[string]interface{}{
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
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.update_status", dao.opts.Table))
	defer span.End()

	sql, args := psql.Update(dao.opts.Table).
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
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.update_error_code", dao.opts.Table))
	defer span.End()

	_, err := dao.update(ctx, id, map[string]interface{}{
		"status":     status,
		"error_code": code,
		"updated_at": sq.Expr("NOW()"),
	})
	return err
}

func (dao *attemptDao) ListUnqueuedForUpdate(ctx context.Context, maxScheduledAt time.Time, limit int) (list []*entities.Attempt, err error) {
	sql := "SELECT * FROM attempts WHERE status = 'INIT' AND created_at <= now() - INTERVAL '30 SECOND' AND scheduled_at < $1 limit $2 FOR UPDATE SKIP LOCKED"
	err = dao.UnsafeDB(ctx).SelectContext(ctx, &list, sql, maxScheduledAt, limit)
	return
}

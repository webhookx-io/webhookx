package dao

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/types"
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
}

func NewAttemptDao(db *sqlx.DB, workspace bool) AttemptDAO {
	return &attemptDao{
		DAO: NewDAO[entities.Attempt]("attempts", db, workspace),
	}
}

func (dao *attemptDao) UpdateDelivery(ctx context.Context, id string, result *AttemptResult) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"request":      result.Request,
		"response":     result.Response,
		"attempted_at": result.AttemptedAt,
		"status":       result.Status,
		"error_code":   result.ErrorCode,
	})
	return err
}

func (dao *attemptDao) UpdateStatus(ctx context.Context, id string, status entities.AttemptStatus) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"status": status,
	})
	return err
}

func (dao *attemptDao) UpdateErrorCode(ctx context.Context, id string, status entities.AttemptStatus, code entities.AttemptErrorCode) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"status":     status,
		"error_code": code,
	})
	return err
}

func (dao *attemptDao) ListUnqueued(ctx context.Context, limit int64) (list []*entities.Attempt, err error) {
	sql := "SELECT * FROM attempts WHERE status = 'INIT' and created_at <= now() - INTERVAL '60 SECOND' limit $1"
	err = dao.UnsafeDB(ctx).SelectContext(ctx, &list, sql, limit)
	return
}

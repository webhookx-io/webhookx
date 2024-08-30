package dao

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
	"time"
)

type attemptDao struct {
	*DAO[entities.Attempt]
}

type DeliveryResult struct {
	Request   *entities.AttemptRequest
	Response  *entities.AttemptResponse
	AttemptAt time.Time
	Status    entities.AttemptStatus
	ErrorCode *entities.AttemptErrorCode
}

func (dao *attemptDao) UpdateDelivery(ctx context.Context, id string, result *DeliveryResult) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"request":    result.Request,
		"response":   result.Response,
		"attempt_at": result.AttemptAt.Unix(),
		"status":     result.Status,
		"error_code": result.ErrorCode,
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

func NewAttemptDao(db *sqlx.DB, workspace bool) AttemptDAO {
	return &attemptDao{
		DAO: NewDAO[entities.Attempt]("attempts", db, workspace),
	}
}

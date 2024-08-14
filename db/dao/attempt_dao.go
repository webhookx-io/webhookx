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

func (dao *attemptDao) UpdateDelivery(
	ctx context.Context,
	id string,
	request *entities.AttemptRequest,
	response *entities.AttemptResponse,
	attemptAt time.Time,
	status entities.AttemptStatus) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"request":    request,
		"response":   response,
		"attempt_at": attemptAt.Unix(),
		"status":     status,
	})
	return err
}

func (dao *attemptDao) UpdateStatus(ctx context.Context, id string, status entities.AttemptStatus) error {
	_, err := dao.update(ctx, id, map[string]interface{}{
		"status": status,
	})
	return err
}

func NewAttemptDao(db *sqlx.DB, workspace bool) AttemptDAO {
	return &attemptDao{
		DAO: NewDAO[entities.Attempt]("attempts", db, workspace),
	}
}

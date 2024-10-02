package dao

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
)

type attemptDetailDao struct {
	*DAO[entities.AttemptDetail]
}

func NewAttemptDetailDao(db *sqlx.DB, workspace bool) AttemptDetailDAO {
	return &attemptDetailDao{
		DAO: NewDAO[entities.AttemptDetail]("attempt_details", db, workspace),
	}
}

func (dao *attemptDetailDao) Upsert(ctx context.Context, attemptDetail *entities.AttemptDetail) error {
	now := time.Now()
	values := []interface{}{attemptDetail.ID, attemptDetail.RequestHeaders, attemptDetail.RequestBody, attemptDetail.ResponseHeaders, attemptDetail.ResponseBody, now, now, attemptDetail.WorkspaceId}

	sql := `INSERT INTO attempt_details (id, request_headers, request_body, response_headers, response_body, created_at, updated_at, ws_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
		ON CONFLICT (id) DO UPDATE SET 
		request_headers = EXCLUDED.request_headers, 
		request_body = EXCLUDED.request_body, 
		response_headers = EXCLUDED.response_headers, 
		response_body = EXCLUDED.response_body, 
		updated_at = EXCLUDED.updated_at`

	result, err := dao.DB(ctx).ExecContext(ctx, sql, values...)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	return err
}

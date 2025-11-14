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
		CacheName:      constants.AttemptDetailCacheKey.Name,
	}
	return &attemptDetailDao{
		DAO: NewDAO[entities.AttemptDetail](db, bus, opts),
	}
}

func (dao *attemptDetailDao) Insert(ctx context.Context, attemptDetail *entities.AttemptDetail) error {
	ctx, span := tracing.Start(ctx, fmt.Sprintf("dao.%s.insert", dao.opts.Table), trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

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

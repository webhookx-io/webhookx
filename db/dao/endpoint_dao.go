package dao

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type endpointDAO struct {
	*DAO[entities.Endpoint]
}

func NewEndpointDAO(db *sqlx.DB, fns ...OptionFunc) EndpointDAO {
	opts := Options{
		Table:          "endpoints",
		EntityName:     "endpoint",
		CachePropagate: true,
		CacheName:      constants.EndpointCacheKey.Name,
	}
	for _, fn := range fns {
		fn(&opts)
	}
	return &endpointDAO{
		DAO: NewDAO[entities.Endpoint](db, opts),
	}
}

func (dao *endpointDAO) Disable(ctx context.Context, id string) (bool, error) {
	ctx, span := dao.trace(ctx, fmt.Sprintf("dao.%s.disable", dao.opts.Table))
	defer span.End()

	v, err := dao.updateOne(ctx, id, map[string]interface{}{
		"enabled":    false,
		"updated_at": sq.Expr("NOW()"),
	}, map[string]interface{}{
		"enabled": true,
	})
	if err != nil {
		return false, err
	}
	return v != nil, nil
}

type EndpointQuery struct {
	Query
	Enabled     *bool
	WorkspaceId *string
}

func (q *EndpointQuery) ToQuery() *Query {
	query := q.clone()
	if q.Enabled != nil {
		query.Where("enabled", Equal, *q.Enabled)
	}
	if q.WorkspaceId != nil {
		query.Where("ws_id", Equal, *q.WorkspaceId)
	}
	return query
}

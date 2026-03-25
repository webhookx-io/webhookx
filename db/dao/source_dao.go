package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type sourceDAO struct {
	*DAO[entities.Source]
}

func NewSourceDAO(db *sqlx.DB, fns ...OptionFunc) SourceDAO {
	opts := Options{
		Table:          "sources",
		EntityName:     "source",
		CachePropagate: true,
		CacheName:      constants.SourceCacheKey.Name,
	}
	return &sourceDAO{
		DAO: NewDAO[entities.Source](db, opts, fns...),
	}
}

type SourceQuery struct {
	Query

	WorkspaceId *string
}

func (q *SourceQuery) ToQuery() *Query {
	query := q.clone()
	if q.WorkspaceId != nil {
		query.Where("ws_id", Equal, *q.WorkspaceId)
	}
	return &query
}

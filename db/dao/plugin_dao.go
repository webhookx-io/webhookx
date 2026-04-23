package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type pluginDAO struct {
	*DAO[entities.Plugin]
}

func NewPluginDAO(db *sqlx.DB, fns ...OptionFunc) PluginDAO {
	opts := Options{
		Table:          "plugins",
		EntityName:     "plugin",
		CachePropagate: true,
		CacheName:      constants.PluginCacheKey.Name,
	}
	for _, fn := range fns {
		fn(&opts)
	}
	return &pluginDAO{
		DAO: NewDAO[entities.Plugin](db, opts),
	}
}

type PluginQuery struct {
	Query

	WorkspaceId *string
	EndpointId  *string
	SourceId    *string
	Enabled     *bool
}

func (q *PluginQuery) ToQuery() *Query {
	query := q.clone()
	if q.WorkspaceId != nil {
		query.Where("ws_id", Equal, *q.WorkspaceId)
	}
	if q.EndpointId != nil {
		query.Where("endpoint_id", Equal, *q.EndpointId)
	}
	if q.SourceId != nil {
		query.Where("source_id", Equal, *q.SourceId)
	}
	if q.Enabled != nil {
		query.Where("enabled", Equal, *q.Enabled)
	}
	return &query
}

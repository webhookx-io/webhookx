package dao

import (
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
	return &endpointDAO{
		DAO: NewDAO[entities.Endpoint](db, opts, fns...),
	}
}

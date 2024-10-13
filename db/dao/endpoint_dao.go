package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type endpointDAO struct {
	*DAO[entities.Endpoint]
}

func NewEndpointDAO(db *sqlx.DB, workspace bool) EndpointDAO {
	opts := Options{
		Table:          "endpoints",
		EntityName:     "endpoint",
		Workspace:      workspace,
		CachePropagate: true,
		CacheKey:       constants.EndpointCacheKey,
	}
	return &endpointDAO{
		DAO: NewDAO[entities.Endpoint](db, opts),
	}
}

package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
)

type endpointDAO struct {
	*DAO[entities.Endpoint]
}

func NewEndpointDAO(db *sqlx.DB) EndpointDAO {
	return &endpointDAO{
		DAO: NewDAO[entities.Endpoint]("endpoints", db),
	}
}

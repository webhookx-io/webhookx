package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
)

type eventDao struct {
	*DAO[entities.Event]
}

func NewEventDao(db *sqlx.DB) EventDAO {
	return &eventDao{
		DAO: NewDAO[entities.Event]("events", db),
	}
}

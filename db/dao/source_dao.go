package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/eventbus"
)

type sourceDAO struct {
	*DAO[entities.Source]
}

func NewSourceDAO(db *sqlx.DB, bus *eventbus.EventBus, workspace bool) SourceDAO {
	opts := Options{
		Table:          "sources",
		EntityName:     "source",
		Workspace:      workspace,
		CachePropagate: true,
		CacheKey:       constants.SourceCacheKey,
	}
	return &sourceDAO{
		DAO: NewDAO[entities.Source](db, bus, opts),
	}
}

package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type sourceDAO struct {
	*DAO[entities.Source]
}

func NewSourceDAO(db *sqlx.DB, workspace bool) SourceDAO {
	opts := Options{
		Table:          "sources",
		EntityName:     "Source",
		Workspace:      workspace,
		CachePropagate: false,
		CacheKey:       constants.SourceCacheKey,
	}
	return &sourceDAO{
		DAO: NewDAO[entities.Source](db, opts),
	}
}

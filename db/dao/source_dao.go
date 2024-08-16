package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
)

type sourceDAO struct {
	*DAO[entities.Source]
}

func NewSourceDAO(db *sqlx.DB, workspace bool) SourceDAO {
	return &sourceDAO{
		DAO: NewDAO[entities.Source]("sources", db, workspace),
	}
}

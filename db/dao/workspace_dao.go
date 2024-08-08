package dao

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
)

type workspaceDAO struct {
	*DAO[entities.Workspace]
}

func NewWorkspaceDAO(db *sqlx.DB) WorkspaceDAO {
	return &workspaceDAO{
		DAO: NewDAO[entities.Workspace]("workspaces", db, false),
	}
}

func (dao *workspaceDAO) GetDefault(ctx context.Context) (*entities.Workspace, error) {
	return dao.selectByField(ctx, "name", "default")
}

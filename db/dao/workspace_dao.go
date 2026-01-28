package dao

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type workspaceDAO struct {
	*DAO[entities.Workspace]
}

func NewWorkspaceDAO(db *sqlx.DB, fns ...OptionFunc) WorkspaceDAO {
	opts := Options{
		Table:          "workspaces",
		EntityName:     "workspace",
		CachePropagate: true,
		CacheName:      constants.WorkspaceCacheKey.Name,
	}
	return &workspaceDAO{
		DAO: NewDAO[entities.Workspace](db, opts, fns...),
	}
}

func (dao *workspaceDAO) GetDefault(ctx context.Context) (*entities.Workspace, error) {
	return dao.selectByField(ctx, "name", "default")
}

func (dao *workspaceDAO) GetWorkspace(ctx context.Context, name string) (*entities.Workspace, error) {
	return dao.selectByField(ctx, "name", name)
}

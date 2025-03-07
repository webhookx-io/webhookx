package dao

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/eventbus"
)

type workspaceDAO struct {
	*DAO[entities.Workspace]
}

func NewWorkspaceDAO(db *sqlx.DB, bus *eventbus.EventBus) WorkspaceDAO {
	opts := Options{
		Table:          "workspaces",
		EntityName:     "workspace",
		Workspace:      false,
		CachePropagate: true,
		CacheKey:       constants.WorkspaceCacheKey,
	}
	return &workspaceDAO{
		DAO: NewDAO[entities.Workspace](db, bus, opts),
	}
}

func (dao *workspaceDAO) GetDefault(ctx context.Context) (*entities.Workspace, error) {
	return dao.selectByField(ctx, "name", "default")
}

func (dao *workspaceDAO) GetWorkspace(ctx context.Context, name string) (*entities.Workspace, error) {
	return dao.selectByField(ctx, "name", name)
}

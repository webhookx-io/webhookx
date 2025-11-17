package dao

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/utils"
)

type pluginDAO struct {
	*DAO[entities.Plugin]
}

func NewPluginDAO(db *sqlx.DB, bus *eventbus.EventBus, workspace bool) PluginDAO {
	opts := Options{
		Table:          "plugins",
		EntityName:     "plugin",
		Workspace:      workspace,
		CachePropagate: true,
		CacheName:      constants.PluginCacheKey.Name,
	}
	return &pluginDAO{
		DAO: NewDAO[entities.Plugin](db, bus, opts),
	}
}

func (dao *pluginDAO) ListEndpointPlugin(ctx context.Context, endpointId string) ([]*entities.Plugin, error) {
	q := query.PluginQuery{}
	q.EndpointId = &endpointId
	q.Enabled = utils.Pointer(true)
	return dao.List(ctx, &q)
}

func (dao *pluginDAO) ListSourcePlugin(ctx context.Context, sourceId string) ([]*entities.Plugin, error) {
	q := query.PluginQuery{}
	q.SourceId = &sourceId
	q.Enabled = utils.Pointer(true)
	return dao.List(ctx, &q)
}

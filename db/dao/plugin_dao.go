package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/eventbus"
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

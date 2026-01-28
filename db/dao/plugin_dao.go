package dao

import (
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db/entities"
)

type pluginDAO struct {
	*DAO[entities.Plugin]
}

func NewPluginDAO(db *sqlx.DB, fns ...OptionFunc) PluginDAO {
	opts := Options{
		Table:          "plugins",
		EntityName:     "plugin",
		CachePropagate: true,
		CacheName:      constants.PluginCacheKey.Name,
	}
	return &pluginDAO{
		DAO: NewDAO[entities.Plugin](db, opts, fns...),
	}
}

package dao

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/utils"
)

type pluginDAO struct {
	*DAO[entities.Plugin]
}

func NewPluginDAO(db *sqlx.DB, workspace bool) PluginDAO {
	return &pluginDAO{
		DAO: NewDAO[entities.Plugin]("plugins", db, workspace),
	}
}

func (dao *pluginDAO) ListEndpointPlugin(ctx context.Context, endpointId string) ([]*entities.Plugin, error) {
	q := query.PluginQuery{}
	q.EndpointId = &endpointId
	q.Enabled = utils.Pointer(true)
	return dao.List(ctx, &q)
}

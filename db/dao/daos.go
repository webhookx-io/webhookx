package dao

import (
	"context"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
)

type BaseDAO[T any] interface {
	Get(ctx context.Context, id string) (*T, error)
	Select(ctx context.Context, field string, id string) (*T, error)
	Insert(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Upsert(ctx context.Context, fields []string, entity *T) error
	Delete(ctx context.Context, id string) (bool, error)
	Page(ctx context.Context, q query.Queryer) ([]*T, int64, error)
	List(ctx context.Context, q query.Queryer) ([]*T, error)
	Count(ctx context.Context, conditions map[string]interface{}) (int64, error)
	BatchInsert(ctx context.Context, entities []*T) error
}

type WorkspaceDAO interface {
	BaseDAO[entities.Workspace]
	GetDefault(ctx context.Context) (*entities.Workspace, error)
	GetWorkspace(ctx context.Context, name string) (*entities.Workspace, error)
}

type EndpointDAO interface {
	BaseDAO[entities.Endpoint]
}

type EventDAO interface {
	BaseDAO[entities.Event]
	BatchInsertIgnoreConflict(ctx context.Context, events []*entities.Event) ([]string, error)
}

type AttemptDAO interface {
	BaseDAO[entities.Attempt]
	UpdateStatusToQueued(ctx context.Context, ids []string) error
	UpdateErrorCode(ctx context.Context, id string, status entities.AttemptStatus, code entities.AttemptErrorCode) error
	UpdateDelivery(ctx context.Context, id string, result *AttemptResult) error
	ListUnqueuedForUpdate(ctx context.Context, limit int) (list []*entities.Attempt, err error)
}

type SourceDAO interface {
	BaseDAO[entities.Source]
}

type AttemptDetailDAO interface {
	BaseDAO[entities.AttemptDetail]
	Insert(ctx context.Context, attemptDetail *entities.AttemptDetail) error
}

type PluginDAO interface {
	BaseDAO[entities.Plugin]
	ListEndpointPlugin(ctx context.Context, endpointId string) (list []*entities.Plugin, err error)
	ListSourcePlugin(ctx context.Context, sourceId string) ([]*entities.Plugin, error)
}

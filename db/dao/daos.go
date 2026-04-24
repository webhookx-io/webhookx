package dao

import (
	"context"
	"time"

	"github.com/webhookx-io/webhookx/db/entities"
)

type BaseDAO[T any] interface {
	Get(ctx context.Context, id string) (*T, error)
	Select(ctx context.Context, field string, id string) (*T, error)
	Insert(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Upsert(ctx context.Context, fields []string, entity *T) error
	Delete(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, query *Query) (int64, error)
	List(ctx context.Context, query *Query) ([]*T, error)
	Cursor(ctx context.Context, query *Query) (Cursor[*T], error)
	BatchInsert(ctx context.Context, entities []*T) error
}

type WorkspaceDAO interface {
	BaseDAO[entities.Workspace]
	GetDefault(ctx context.Context) (*entities.Workspace, error)
	GetWorkspace(ctx context.Context, name string) (*entities.Workspace, error)
}

type EndpointDAO interface {
	BaseDAO[entities.Endpoint]
	Disable(ctx context.Context, id string) error
}

type EventDAO interface {
	BaseDAO[entities.Event]
	BatchInsertIgnoreConflict(ctx context.Context, events []*entities.Event) ([]string, error)
	ListExistingUniqueIDs(ctx context.Context, uniques []string) ([]string, error)
}

type AttemptDAO interface {
	BaseDAO[entities.Attempt]
	UpdateStatusToQueued(ctx context.Context, ids []string) error
	UpdateErrorCode(ctx context.Context, id string, status entities.AttemptStatus, code entities.AttemptErrorCode) error
	UpdateDelivery(ctx context.Context, result *AttemptResult) error
	ListUnqueuedForUpdate(ctx context.Context, maxScheduledAt time.Time, limit int) (list []*entities.Attempt, err error)
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
}

package dao

import (
	"context"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"time"
)

type BaseDAO[T any] interface {
	Get(ctx context.Context, id string) (*T, error)
	Insert(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) (bool, error)
	Page(ctx context.Context, q query.DatabaseQuery) ([]*T, int64, error)
	List(ctx context.Context, q query.DatabaseQuery) ([]*T, error)
	BatchInsert(ctx context.Context, entities []*T) error
}

type WorkspaceDAO interface {
	BaseDAO[entities.Workspace]
	GetDefault(ctx context.Context) (*entities.Workspace, error)
}

type EndpointDAO interface {
	BaseDAO[entities.Endpoint]
}

type EventDAO interface {
	BaseDAO[entities.Event]
}

type AttemptDAO interface {
	BaseDAO[entities.Attempt]
	UpdateStatus(ctx context.Context, id string, status entities.AttemptStatus) error
	UpdateDelivery(ctx context.Context, id string, request *entities.AttemptRequest, response *entities.AttemptResponse, attemptAt time.Time, status entities.AttemptStatus) error
}

type SourceDAO interface {
	BaseDAO[entities.Source]
}

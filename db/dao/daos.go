package dao

import (
	"context"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
)

type BaseDAO[T any] interface {
	Get(ctx context.Context, id string) (*T, error)
	Insert(ctx context.Context, entity *T) error
	Update(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id string) (bool, error)
	Page(ctx context.Context, q query.Queryer) ([]*T, int64, error)
	List(ctx context.Context, q query.Queryer) ([]*T, error)
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
	UpdateErrorCode(ctx context.Context, id string, status entities.AttemptStatus, code entities.AttemptErrorCode) error
	UpdateDelivery(ctx context.Context, id string, result *AttemptResult) error
	ListUnqueued(ctx context.Context, limit int64) (list []*entities.Attempt, err error)
}

type SourceDAO interface {
	BaseDAO[entities.Source]
}

type AttemptDetailDAO interface {
	BaseDAO[entities.AttemptDetail]
	Upsert(ctx context.Context, attemptDetail *entities.AttemptDetail) error
}

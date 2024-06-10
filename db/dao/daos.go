package dao

import (
	"context"
	"github.com/webhookx-io/webhookx/db/query"
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

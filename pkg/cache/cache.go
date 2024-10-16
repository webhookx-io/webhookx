package cache

import (
	"context"
	"time"
)

type Cache interface {
	Put(ctx context.Context, key string, val interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, val interface{}) (exist bool, err error)
	Remove(ctx context.Context, key string) error
	Exist(ctx context.Context, key string) (bool, error)
}

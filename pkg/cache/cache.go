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

type Options struct {
	Expiration time.Duration
}

// Get gets value from the cache, load from callback function if cache value does not exist.
// Example:
//
//	  cache.Get(cacheInstance, ctx, "workspaces:uid", func(ctx context.Context) (*entities.Workspace, error) {
//		   return db.Workspace.Get(ctx, workspaceId)
//	  }, nil)
func Get[T any](cache Cache, ctx context.Context, key string, callback func(ctx context.Context) (*T, error), options *Options) (*T, error) {
	value := new(T)
	exist, err := cache.Get(ctx, key, value)
	if err != nil {
		return nil, err
	}
	if exist {
		return value, nil
	}

	value, err = callback(ctx)
	if err != nil {
		return nil, err
	}

	timeout := time.Second * 10 // FIXME: hardcode value
	if options != nil {
		timeout = options.Expiration
	}

	err = cache.Put(ctx, key, value, timeout)
	if err != nil {
		return nil, err
	}

	return value, nil
}

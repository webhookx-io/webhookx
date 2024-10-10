package mcache

import (
	"context"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"sync/atomic"
	"time"
)

var globalMcache atomic.Value

func Set(mcache *MCache) {
	globalMcache.Store(mcache)
}

// MCache is multiple layer cache
type MCache struct {
	lru   *expirable.LRU[string, any]
	cache cache.Cache
}

func NewMCache(cache cache.Cache) *MCache {
	return &MCache{
		cache: cache,
		lru:   expirable.NewLRU[string, any](1000, nil, time.Second*10),
	}
}

func (c *MCache) InvalidateLocal(key string) {
	c.lru.Remove(key)
}

func (c *MCache) Invalidate(ctx context.Context, key string) error {
	err := c.cache.Remove(ctx, key)
	if err != nil {
		return err
	}
	c.InvalidateLocal(key)
	return nil
}

type Callback[T any] func(ctx context.Context, id string) (*T, error)

func Delete(ctx context.Context, key string) error {
	mcache := globalMcache.Load().(*MCache)
	return mcache.Invalidate(ctx, key)
}

func Load[T any](ctx context.Context, key string, cb Callback[T], id string) (*T, error) {
	mcache := globalMcache.Load().(*MCache)

	// L1 cache look up
	v, ok := mcache.lru.Get(key)
	if ok {
		return v.(*T), nil
	}

	// L2 cache look up
	value := new(T)
	exist, err := mcache.cache.Get(ctx, key, value)
	if err != nil {
		return nil, err
	}
	if exist {
		mcache.lru.Add(key, value)
		return value, nil
	}

	// TODO dog-piled
	value, err = cb(ctx, id)
	if err != nil || value == nil {
		return value, err
	}

	expiration := constants.CacheDefaultExpiration
	err = mcache.cache.Put(ctx, key, value, expiration)
	if err != nil {
		return nil, err
	}

	mcache.lru.Add(key, value)

	return value, nil
}

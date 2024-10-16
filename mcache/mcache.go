package mcache

import (
	"context"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"go.uber.org/zap"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultL1TTL = time.Second * 10
	DefaultL2TTL = time.Second * 60
)

// shared instance
var globalMcache atomic.Value

func Set(mcache *MCache) {
	globalMcache.Store(mcache)
}

// MCache is multiple levels cache
type MCache struct {
	mux sync.Mutex
	l1  *expirable.LRU[string, any]
	l2  cache.Cache
}

type Options struct {
	L1Size int
	L1TTL  time.Duration
	L2     cache.Cache
}

func NewMCache(opts *Options) *MCache {
	return &MCache{
		l1: expirable.NewLRU[string, any](opts.L1Size, nil, opts.L1TTL),
		l2: opts.L2,
	}
}

func (c *MCache) InvalidateL1(ctx context.Context, key string) error {
	c.l1.Remove(key)
	return nil
}

func (c *MCache) InvalidateL2(ctx context.Context, key string) error {
	return c.l2.Remove(ctx, key)
}

func (c *MCache) Invalidate(ctx context.Context, key string) error {
	zap.S().Debugf("invalidating cache %s", key)
	if err := c.InvalidateL2(ctx, key); err != nil {
		return err
	}
	if err := c.InvalidateL1(ctx, key); err != nil {
		return err
	}
	return nil
}

func Invalidate(ctx context.Context, key string) error {
	mcache := globalMcache.Load().(*MCache)
	return mcache.Invalidate(ctx, key)
}

type Callback[T any] func(ctx context.Context, id string) (*T, error)

type LoadOptions struct {
	DisableLRU bool
}

var defaultOpts LoadOptions

func Load[T any](ctx context.Context, key string, opts *LoadOptions, cb Callback[T], id string) (*T, error) {
	mcache := globalMcache.Load().(*MCache)
	if opts == nil {
		opts = &defaultOpts
	}

	if !opts.DisableLRU {
		// L1 looks up
		v, ok := mcache.l1.Get(key)
		if ok {
			return v.(*T), nil
		}
	}

	// L2 looks up
	value := new(T)
	exist, err := mcache.l2.Get(ctx, key, value)
	if err != nil {
		return nil, err
	}
	if exist {
		mcache.l1.Add(key, value)
		return value, nil
	}

	// L3(IO/DB) looks up
	// mutex to prevent dog-pile effects
	mcache.mux.Lock()
	defer mcache.mux.Unlock()
	if v, ok := mcache.l1.Get(key); ok {
		return v.(*T), nil
	}
	value, err = cb(ctx, id)
	if err != nil || value == nil {
		return value, err
	}

	err = mcache.l2.Put(ctx, key, value, DefaultL2TTL)
	if err != nil {
		return nil, err
	}

	if !opts.DisableLRU {
		mcache.l1.Add(key, value)
	}

	return value, nil
}

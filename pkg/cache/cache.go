package cache

import (
	"context"
	"github.com/webhookx-io/webhookx/config"
	"time"
)

type Cache interface {
	Put(ctx context.Context, key string, val interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, val interface{}) (exist bool, err error)
	Remove(ctx context.Context, key string) error
	Exist(ctx context.Context, key string) (bool, error)
}

func New(config config.RedisConfig) Cache {
	client := config.GetClient()
	return NewRedisCache(client)
}

type Options struct {
	Timeout time.Duration
}

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

	timeout := time.Second * 10 // todo: default value
	if options != nil {
		timeout = options.Timeout
	}

	err = cache.Put(ctx, key, value, timeout)
	if err != nil {
		return nil, err
	}

	return value, nil
}

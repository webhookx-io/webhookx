package cache

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/pkg/serializer"
	"time"
)

type RedisCache struct {
	c *redis.Client
	s serializer.Serializer
}

func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{
		c: client,
		s: serializer.Gob,
	}
}

func (s *RedisCache) Put(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if value == nil {
		return nil
	}
	b, err := s.s.Serialize(value)
	if err != nil {
		return err
	}
	return s.c.Set(ctx, key, b, expiration).Err()
}

func (s *RedisCache) Get(ctx context.Context, key string, value interface{}) (exist bool, err error) {
	result, err := s.c.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return true, s.s.Deserialize(result, value)
}

func (s *RedisCache) Remove(ctx context.Context, key string) error {
	return s.c.Del(ctx, key).Err()
}

func (s *RedisCache) Exist(ctx context.Context, key string) (bool, error) {
	result, err := s.c.Exists(ctx, key).Result()
	return result == 1, err
}

package distributed

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLocker struct {
	client *redis.Client
}

func NewRedisLocker(client *redis.Client) *RedisLocker {
	return &RedisLocker{
		client: client,
	}
}

func (l *RedisLocker) TryLock(ctx context.Context, option LockOption) (bool, error) {
	key := fmt.Sprintf("webhookx:lock:%s", option.Name)
	acquired, err := l.client.SetNX(ctx, key, time.Now(), option.TTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	return acquired, nil
}

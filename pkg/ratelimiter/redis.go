package ratelimiter

import (
	"context"
	"github.com/go-redis/redis_rate/v10"
	"time"
)

type RedisLimiter struct {
	limiter *redis_rate.Limiter
}

func NewRedisLimiter(limiter *redis_rate.Limiter) *RedisLimiter {
	return &RedisLimiter{
		limiter: limiter,
	}
}

func (rl *RedisLimiter) Allow(ctx context.Context, key string, quota int, duration time.Duration) (Result, error) {
	res := Result{}
	limit := redis_rate.Limit{
		Rate:   quota,
		Burst:  quota,
		Period: duration,
	}
	r, err := rl.limiter.Allow(ctx, key, limit)
	if err != nil {
		return res, err
	}
	res.Allowed = r.Allowed > 0
	res.Remaining = r.Remaining
	res.Reset = r.ResetAfter
	if r.RetryAfter != -1 {
		res.RetryAfter = r.RetryAfter
	}
	return res, nil
}

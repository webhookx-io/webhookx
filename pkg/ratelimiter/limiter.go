package ratelimiter

import (
	"context"
	"time"
)

type Result struct {
	Allowed    bool
	Remaining  int
	Reset      time.Duration
	RetryAfter time.Duration
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, quota int, duration time.Duration) (Result, error)
}

package retry

import (
	"time"
)

type Strategy string

const (
	FixedStrategy   Strategy = "fixed"
	BackoffStrategy Strategy = "backoff"
)

const Stop time.Duration = -1

type Retry interface {
	NextDelay(attempts int) time.Duration
}

type Option func(Retry)

func NewRetry(strategy Strategy, opts ...Option) Retry {
	var retry Retry
	switch strategy {
	case FixedStrategy:
		retry = newFixedStrategyRetry()
	case BackoffStrategy:
		panic("implement me")
	default:
		panic("invalid strategy: " + strategy)
	}
	for _, opt := range opts {
		opt(retry)
	}
	return retry
}

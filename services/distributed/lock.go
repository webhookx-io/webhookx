package distributed

import (
	"context"
	"time"
)

type LockOption struct {
	Name string
	TTL  time.Duration
}

type Locker interface {
	TryLock(ctx context.Context, option LockOption) (acquired bool, err error)
}

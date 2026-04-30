package distributed

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Distributed struct {
	log    *zap.SugaredLogger
	locker Locker
}

type Option func(*Distributed)

func WithLogger(logger *zap.SugaredLogger) Option { return func(m *Distributed) { m.log = logger } }

func WithLocker(locker Locker) Option { return func(m *Distributed) { m.locker = locker } }

func NewDistributed(opts ...Option) *Distributed {
	distributed := &Distributed{}
	for _, opt := range opts {
		opt(distributed)
	}
	return distributed
}

func (d *Distributed) Mutex(ctx context.Context, option LockOption, fn func(ctx context.Context) error) error {
	// verbose
	// d.log.Debugf("acquiring lock '%s' (ttl=%s)", option.Name, option.TTL.String())

	acquired, err := d.locker.TryLock(ctx, option)
	if err != nil {
		return fmt.Errorf("lock acquire error: %w", err)
	}
	if !acquired {
		// verbose
		// d.log.Debugf("lock not acquired: '%s'", option.Name)
		return nil
	}

	return fn(ctx)
}

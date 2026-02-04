package pool

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrPoolTernimated = errors.New("pool is ternimated")
	ErrTimeout        = errors.New("timeout")
)

type item[T any] struct {
	ctx   context.Context
	value T
}

type Handler[T any] interface {
	Handle(ctx context.Context, value T)
}

type HandlerFunc[T any] func(ctx context.Context, value T)

func (fn HandlerFunc[T]) Handle(ctx context.Context, value T) {
	fn(ctx, value)
}

type Pool[T any] struct {
	ctx     context.Context
	cancel  context.CancelFunc
	queue   chan item[T]
	handler Handler[T]
	wg      sync.WaitGroup
}

func New[T any](size int, consumers int, handler Handler[T]) *Pool[T] {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool[T]{
		ctx:     ctx,
		cancel:  cancel,
		queue:   make(chan item[T], size),
		handler: handler,
	}

	for i := 0; i < consumers; i++ {
		pool.wg.Go(pool.consume)
	}

	return pool
}

func (p *Pool[T]) Submit(ctx context.Context, timeout time.Duration, value T) error {
	if p.ctx.Err() != nil {
		return ErrPoolTernimated
	}

	it := item[T]{
		ctx:   ctx,
		value: value,
	}

	select {
	case <-p.ctx.Done():
		return ErrPoolTernimated
	case p.queue <- it:
		return nil
	case <-time.After(timeout):
		return ErrTimeout
	}
}

func (p *Pool[T]) consume() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case t := <-p.queue:
			p.handler.Handle(t.ctx, t.value)
		}
	}
}

func (p *Pool[T]) Shutdown() {
	p.cancel()
	p.wg.Wait()
}

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

type Pool struct {
	ctx    context.Context
	cancel context.CancelFunc

	workers int

	tasks chan Task
	wait  sync.WaitGroup
}

func NewPool(size int, workers int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		ctx:     ctx,
		cancel:  cancel,
		workers: workers,
		tasks:   make(chan Task, size),
	}

	pool.wait.Add(workers)

	for i := 0; i < workers; i++ {
		go pool.consume()
	}

	return pool
}

func (p *Pool) SubmitFn(timeout time.Duration, fn func()) error {
	if fn == nil {
		return errors.New("fn is nil")
	}

	taks := &task{
		fn: fn,
	}
	return p.Submit(timeout, taks)
}

func (p *Pool) Submit(timeout time.Duration, task Task) error {
	if task == nil {
		return errors.New("task is nil")
	}

	if p.ctx.Err() != nil {
		return ErrPoolTernimated
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case p.tasks <- task:
		return nil
	case <-timer.C:
		return ErrTimeout
	}
}

func (p *Pool) consume() {
	defer p.wait.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case t := <-p.tasks:
			t.Execute()
		}
	}
}

func (p *Pool) Shutdown() {
	if err := p.ctx.Err(); err != nil {
		return
	}

	p.cancel()
	p.wait.Wait()
}

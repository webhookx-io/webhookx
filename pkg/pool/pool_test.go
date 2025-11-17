package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	pool := NewPool(0, 1)

	err := pool.SubmitFn(time.Second, nil)
	assert.Equal(t, "fn is nil", err.Error())

	err = pool.Submit(time.Second, nil)
	assert.Equal(t, "task is nil", err.Error())

	// panic should be recovered
	err = pool.SubmitFn(time.Second, func() {
		panic("foo")
	})
	assert.NoError(t, err)

	pool.Shutdown()
	pool.Shutdown() // no panic
}

func TestSubmit(t *testing.T) {
	pool := NewPool(5, 1)
	wait := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		wait.Add(1)
		err := pool.SubmitFn(time.Second, func() {
			wait.Done()
		})
		assert.NoError(t, err)
	}
	wait.Wait()
}

func TestSubmitWithTimeout(t *testing.T) {
	pool := NewPool(1, 1)
	err := pool.SubmitFn(time.Second, func() {
		time.Sleep(time.Second * 5)
	})
	assert.NoError(t, err)
	err = pool.SubmitFn(time.Second, func() {
		time.Sleep(time.Second * 5)
	})
	assert.NoError(t, err)
	err = pool.SubmitFn(time.Second, func() {})
	assert.Equal(t, ErrTimeout, err)
}

func TestShutdown(t *testing.T) {
	pool := NewPool(1, 1)
	pool.Shutdown()
	err := pool.SubmitFn(time.Second, func() {})
	assert.Equal(t, ErrPoolTernimated, err)
}

func TestGracefulShutdown(t *testing.T) {
	var counter atomic.Int64

	pool := NewPool(100, 100)

	wait := sync.WaitGroup{}
	wait.Add(100)
	for i := 0; i < 100; i++ {
		err := pool.SubmitFn(time.Second, func() {
			wait.Done()
			time.Sleep(time.Second)
			counter.Add(1)
		})
		assert.NoError(t, err)
	}
	wait.Wait() // wait for all tasks to be scheduled

	pool.Shutdown()
	assert.EqualValues(t, 100, counter.Load()) // all submitted and scheduled tasks should be executed successfully
}

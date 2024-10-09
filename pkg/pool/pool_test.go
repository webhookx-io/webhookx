package pool

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
	var counter int64
	atomic.StoreInt64(&counter, 0)

	pool := NewPool(100, 100)

	for i := 0; i < 100; i++ {
		err := pool.SubmitFn(time.Second, func() {
			time.Sleep(time.Second)
			atomic.AddInt64(&counter, 1)
		})
		assert.NoError(t, err)
	}

	pool.Shutdown()
	assert.EqualValues(t, 100, counter) // all submitted and scheduled tasks should be executed successfully
}

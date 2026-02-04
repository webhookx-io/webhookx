package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var NoopHandler = HandlerFunc[any](func(_ context.Context, _ any) {})

func Test(t *testing.T) {
	t.Run("should return ErrPoolTernimated", func(t *testing.T) {
		pool := New(1, 1, NoopHandler)
		pool.Shutdown()
		err := pool.Submit(context.TODO(), time.Second, nil)
		assert.Equal(t, ErrPoolTernimated, err)
	})

	t.Run("should return ErrTimeout", func(t *testing.T) {
		pool := New(1, 1, HandlerFunc[any](func(_ context.Context, _ any) { time.Sleep(time.Second * 1) }))

		err := pool.Submit(context.TODO(), time.Millisecond*100, nil)
		assert.NoError(t, err)

		err = pool.Submit(context.TODO(), time.Millisecond*100, nil)
		assert.NoError(t, err)

		err = pool.Submit(context.TODO(), time.Millisecond*100, nil)
		assert.Equal(t, ErrTimeout, err)
	})

	t.Run("running tasks should be executed after shutdown", func(t *testing.T) {
		var n atomic.Int64
		wait := sync.WaitGroup{}
		wait.Add(100)
		pool := New(100, 100, HandlerFunc[any](func(ctx context.Context, v any) {
			wait.Done()
			time.Sleep(time.Second)
			n.Add(1)
		}))

		for i := 1; i <= 100; i++ {
			err := pool.Submit(context.TODO(), time.Second, i)
			assert.NoError(t, err)
		}

		wait.Wait() // wait for all tasks to be scheduled

		pool.Shutdown()
		assert.EqualValues(t, 100, n.Load()) // all the scheduled tasks should be executed successfully
	})
}

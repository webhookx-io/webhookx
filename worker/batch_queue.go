package worker

import (
	"sync"
	"time"
)

type BatchQueue[T any] struct {
	MaxBatchSize int
	FlushTimeout time.Duration

	queue chan T
	wg    sync.WaitGroup
}

func NewBatchQueue[T any](size int, batchSize int, flushTimeout time.Duration) *BatchQueue[T] {
	return &BatchQueue[T]{
		queue:        make(chan T, size),
		MaxBatchSize: batchSize,
		FlushTimeout: flushTimeout,
	}
}

func (q *BatchQueue[T]) Add(item T) {
	q.queue <- item
}

func (q *BatchQueue[T]) Close() {
	close(q.queue)
	q.wg.Wait()
}

func (q *BatchQueue[T]) Consume(fn func([]T)) {
	q.wg.Go(func() { q.consume(fn) })
}

func (q *BatchQueue[T]) consume(fn func([]T)) {
	buffer := make([]T, 0, q.MaxBatchSize)
	timeout := time.NewTimer(q.FlushTimeout)
	defer timeout.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		fn(buffer)
		buffer = buffer[:0]
	}

	for {
		select {
		case item, ok := <-q.queue:
			if !ok {
				flush()
				return
			}
			buffer = append(buffer, item)
			if len(buffer) >= q.MaxBatchSize {
				flush()
				timeout.Reset(q.FlushTimeout)
			}
		case <-timeout.C:
			flush()
			timeout.Reset(q.FlushTimeout)
		}
	}
}

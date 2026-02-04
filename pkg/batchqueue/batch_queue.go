package batchqueue

import (
	"context"
	"sync"
	"time"

	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Item[T any] struct {
	value T
	ctx   trace.SpanContext
}

type Handler[T any] func(context.Context, []T)

type BatchQueue[T any] struct {
	Name         string
	MaxBatchSize int
	FlushTimeout time.Duration

	queue chan Item[T]
	wg    sync.WaitGroup
}

func New[T any](name string, size int, batchSize int, flushTimeout time.Duration) *BatchQueue[T] {
	return &BatchQueue[T]{
		Name:         name,
		queue:        make(chan Item[T], size),
		MaxBatchSize: batchSize,
		FlushTimeout: flushTimeout,
	}
}

func (q *BatchQueue[T]) Add(ctx context.Context, item T) {
	ctx, span := tracing.Start(ctx, "queue."+q.Name+".add")
	defer span.End()
	q.queue <- Item[T]{
		ctx:   trace.SpanContextFromContext(ctx),
		value: item,
	}
}

func (q *BatchQueue[T]) Close() {
	close(q.queue)
	q.wg.Wait()
}

func (q *BatchQueue[T]) Consume(handler Handler[T]) {
	q.wg.Go(func() { q.consume(handler) })
}

func (q *BatchQueue[T]) consume(handler Handler[T]) {
	buffer := make([]T, 0, q.MaxBatchSize)
	links := make([]trace.Link, 0, q.MaxBatchSize)
	timeout := time.NewTimer(q.FlushTimeout)
	defer timeout.Stop()

	flush := func() {
		size := len(buffer)
		if size == 0 {
			return
		}

		ctx, span := tracing.Start(context.Background(), "queue."+q.Name+".flush", trace.WithLinks(links...))
		span.SetAttributes(attribute.Int("size", size))
		defer span.End()
		handler(ctx, buffer)
		buffer = buffer[:0]
		links = links[:0]
	}

	for {
		select {
		case item, ok := <-q.queue:
			if !ok {
				flush()
				return
			}
			buffer = append(buffer, item.value)
			if item.ctx.IsValid() {
				links = append(links, trace.Link{SpanContext: item.ctx})
			}
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

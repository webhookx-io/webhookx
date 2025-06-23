package queue

import (
	"context"
	"time"
)

type Message struct {
	ID          string
	Data        []byte
	Time        time.Time
	WorkspaceID string
}

type Options struct {
	Count   int64
	Block   bool
	Timeout time.Duration
}

type Queue interface {
	Enqueue(ctx context.Context, message *Message) error
	Dequeue(ctx context.Context, opts *Options) ([]*Message, error)
	Delete(ctx context.Context, message []*Message) error
	Size(ctx context.Context) (int64, error)
	Stats() map[string]interface{}
}

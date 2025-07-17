package queue

import (
	"context"
	"time"
)

type Message struct {
	Value       []byte
	Time        time.Time
	WorkspaceID string
}

type HandlerFunc func(ctx context.Context, messages []*Message) error

type Queue interface {
	Producer
	Consumer
	Stats() map[string]interface{}
}

type Producer interface {
	Enqueue(ctx context.Context, message *Message) error
}

type Consumer interface {
	StartListen(ctx context.Context, handler HandlerFunc)
}

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

type HandleFunc func(ctx context.Context, messages []*Message) error

type Queue interface {
	Producer
	Consumer
	Stats() map[string]interface{}
}

type Producer interface {
	WriteMessage(ctx context.Context, message *Message) error
}

type Consumer interface {
	StartListen(ctx context.Context, handle HandleFunc)
}

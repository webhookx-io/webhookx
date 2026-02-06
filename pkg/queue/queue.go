package queue

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type Message struct {
	Value        []byte
	Time         time.Time
	WorkspaceID  string
	TraceContext map[string]string
}

func (m *Message) GetTraceContext(ctx context.Context) context.Context {
	if m.TraceContext != nil {
		return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(m.TraceContext))
	}
	return ctx
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

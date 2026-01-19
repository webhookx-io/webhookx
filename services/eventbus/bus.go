package eventbus

import (
	"context"
)

type Handler func(v interface{})

type ClusteringHandler func(v []byte)

type EventBus interface {
	ClusteringBroadcast(ctx context.Context, channel string, value Marshaler) error
	ClusteringSubscribe(channel string, handler ClusteringHandler)
	Broadcast(ctx context.Context, channel string, value interface{})
	Subscribe(channel string, handler Handler)
}

type Marshaler interface {
	Marshal() ([]byte, error)
}

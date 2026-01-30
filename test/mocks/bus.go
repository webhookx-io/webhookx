package mocks

import (
	"context"

	"github.com/webhookx-io/webhookx/services/eventbus"
)

type MockBus struct{}

var _ eventbus.EventBus = &MockBus{}

func (m MockBus) ClusteringBroadcast(ctx context.Context, channel string, value eventbus.Marshaler) error {
	return nil
}

func (m MockBus) ClusteringSubscribe(channel string, handler eventbus.ClusteringHandler) {
}

func (m MockBus) Broadcast(ctx context.Context, channel string, value interface{}) {
}

func (m MockBus) Subscribe(channel string, handler eventbus.Handler) {
}

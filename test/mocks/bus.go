package mocks

import "github.com/webhookx-io/webhookx/eventbus"

type MockBus struct{}

func (m MockBus) ClusteringBroadcast(event string, data interface{}) error {
	return nil
}

func (m MockBus) ClusteringSubscribe(channel string, fn func(data []byte)) {
}

func (m MockBus) Broadcast(channel string, data interface{}) {
}

func (m MockBus) Subscribe(channel string, cb eventbus.Callback) {
}

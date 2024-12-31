package eventbus

import (
	"context"
	"encoding/json"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"sync"
	"time"
)

const channelName = "webhookx"

type EventBus struct {
	ctx      context.Context
	cancel   context.CancelFunc
	nodeID   string
	log      *zap.SugaredLogger
	listener *pq.Listener
	mux      sync.Mutex
	handlers map[string][]func(data []byte)
}

func NewEventBus(nodeID string, dsn string, log *zap.SugaredLogger) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := EventBus{
		ctx:      ctx,
		cancel:   cancel,
		nodeID:   nodeID,
		listener: pq.NewListener(dsn, time.Millisecond*100, time.Second*30, nil),
		mux:      sync.Mutex{},
		handlers: make(map[string][]func(data []byte)),
		log:      log,
	}

	return &bus
}

func (bus *EventBus) Start() error {
	go bus.listenLoop()
	go bus.startListen()
	return nil
}

func (bus *EventBus) startListen() {
	err := bus.listener.Listen(channelName)
	if err != nil {
		bus.log.Errorf("[eventbus] failed to listen on channel %s: %v", channelName, err)
		return
	}
	bus.log.Infof("[eventbus] listening on channel: %s", channelName)
}

func (bus *EventBus) Stop() error {
	bus.cancel()
	return bus.listener.Close()
}

func (bus *EventBus) listenLoop() {
	timeoutDuration := 5 * time.Second
	timeout := time.NewTimer(timeoutDuration)
	for {
		timeout.Reset(timeoutDuration)
		select {
		case <-bus.ctx.Done():
			return
		case n := <-bus.listener.NotificationChannel():
			var payload EventPayload
			if err := json.Unmarshal([]byte(n.Extra), &payload); err != nil {
				bus.log.Errorf("[eventbus] failed to unmarshal payload: %s", err)
				continue
			}
			if payload.Node == bus.nodeID {
				continue
			}
			bus.log.Debugf("[eventbus] received event: channel=%s, payload=%s", n.Channel, n.Extra)
			if handlers, ok := bus.handlers[payload.Event]; ok {
				for _, handler := range handlers {
					handler(payload.Data)
				}
			}
		case <-timeout.C:
			bus.log.Debugf("[eventbus] pinging database")
			err := bus.listener.Ping()
			if err != nil {
				bus.log.Errorf("[eventbus] faield to ping database: %v", err)
			}
		}
	}
}

func (bus *EventBus) Subscribe(channel string, fn func(data []byte)) {
	bus.mux.Lock()
	defer bus.mux.Unlock()

	bus.handlers[channel] = append(bus.handlers[channel], fn)
}

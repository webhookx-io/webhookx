package eventbus

import (
	"context"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/webhookx-io/webhookx/config"
	"go.uber.org/zap"
	"sync"
	"time"
)

type Bus interface {
	Subscribe(channel string, fn func(data []byte))
	Start() error
	Stop() error
}

const channelName = "webhookx"

type DatabaseEventBus struct {
	ctx      context.Context
	cancel   context.CancelFunc
	dsn      string
	log      *zap.SugaredLogger
	mux      sync.Mutex
	handlers map[string][]func(data []byte)
}

func NewDatabaseEventBus(dsn string, log *zap.SugaredLogger) *DatabaseEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := DatabaseEventBus{
		ctx:      ctx,
		cancel:   cancel,
		dsn:      dsn,
		mux:      sync.Mutex{},
		handlers: make(map[string][]func(data []byte)),
		log:      log,
	}

	return &bus
}

func (bus *DatabaseEventBus) Start() error {
	listener := pq.NewListener(bus.dsn, 10*time.Second, time.Minute, nil)
	err := listener.Listen(channelName)
	if err != nil {
		return err
	}
	go bus.listenLoop(listener)
	return nil
}

func (bus *DatabaseEventBus) Stop() error {
	bus.cancel()
	return nil
}

func (bus *DatabaseEventBus) listenLoop(listener *pq.Listener) {
	defer listener.Close()

	bus.log.Infof("[eventbus] listening on channel: %s", channelName)
	for {
		select {
		case <-bus.ctx.Done():
			return
		case n := <-listener.NotificationChannel():
			var payload EventPayload
			if err := json.Unmarshal([]byte(n.Extra), &payload); err != nil {
				bus.log.Errorf("[eventbus] failed to unmarshal payload: %s", err)
				continue
			}
			if payload.Node == config.NODE {
				continue
			}
			bus.log.Debugf("[eventbus] received event: channel=%s, payload=%s", n.Channel, n.Extra)
			if handlers, ok := bus.handlers[payload.Event]; ok {
				for _, handler := range handlers {
					handler(payload.Data)
				}
			}
		case <-time.After(90 * time.Second):
			err := listener.Ping()
			if err != nil {
				bus.log.Errorf("[eventbus] ping error: %v", err)
			}
		}
	}
}

func (bus *DatabaseEventBus) Subscribe(channel string, fn func(data []byte)) {
	bus.mux.Lock()
	defer bus.mux.Unlock()

	bus.handlers[channel] = append(bus.handlers[channel], fn)
}

package eventbus

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/lib/pq"
	"go.uber.org/zap"
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
	bus      evbus.Bus
	db       *sql.DB
}

func NewEventBus(nodeID string, dsn string, log *zap.SugaredLogger, db *sql.DB) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	bus := EventBus{
		ctx:      ctx,
		cancel:   cancel,
		bus:      evbus.New(),
		nodeID:   nodeID,
		listener: pq.NewListener(dsn, time.Millisecond*100, time.Second*30, nil),
		mux:      sync.Mutex{},
		handlers: make(map[string][]func(data []byte)),
		log:      log.Named("eventbus"),
		db:       db,
	}

	return &bus
}

func (bus *EventBus) Start() error {
	go bus.listenClusterLoop()
	go bus.startListen()
	return nil
}

func (bus *EventBus) startListen() {
	err := bus.listener.Listen(channelName)
	if err != nil {
		bus.log.Errorf("failed to listen on channel %s: %v", channelName, err)
		return
	}
	bus.log.Infof(`listening on channel "%s"`, channelName)
}

func (bus *EventBus) Stop() error {
	bus.cancel()
	return bus.listener.Close()
}

func (bus *EventBus) listenClusterLoop() {
	timeoutDuration := 5 * time.Second
	timeout := time.NewTimer(timeoutDuration)
	for {
		timeout.Reset(timeoutDuration)
		select {
		case <-bus.ctx.Done():
			return
		case n := <-bus.listener.NotificationChannel():
			var msg Message
			if err := json.Unmarshal([]byte(n.Extra), &msg); err != nil {
				bus.log.Errorf("failed to unmarshal message: %s", err)
				continue
			}
			if msg.Node == bus.nodeID {
				continue
			}
			bus.log.Debugf("dispatch cluster message: %s", n.Extra)
			if handlers, ok := bus.handlers[msg.Event]; ok {
				for _, handler := range handlers {
					handler(msg.Data)
				}
			}
		case <-timeout.C:
			err := bus.listener.Ping()
			if err != nil {
				bus.log.Errorf("faield to ping database: %v", err)
			}
		}
	}
}

func (bus *EventBus) ClusteringBroadcast(channel string, data interface{}) error {
	bus.bus.Publish(channel, data)

	bytes, err := json.Marshal(data)
	if err != nil {
		bus.log.Errorf("failed to marshal data: %v", err)
		return err
	}
	msg := Message{
		Event: channel,
		Time:  time.Now().UnixMilli(),
		Node:  bus.nodeID,
		Data:  bytes,
	}
	bytes, err = json.Marshal(msg)
	if err != nil {
		bus.log.Errorf("failed to marshal message: %v", err)
		return err
	}

	bus.log.Debugf("broadcasting cluster message: %s", string(bytes))

	statement := fmt.Sprintf("NOTIFY %s, %s", "webhookx", pq.QuoteLiteral(string(bytes)))
	_, err = bus.db.ExecContext(context.TODO(), statement)
	if err != nil {
		bus.log.Errorf("failed to broadcast message: %v", err)
	}
	return err
}

func (bus *EventBus) ClusteringSubscribe(channel string, fn func(data []byte)) {
	bus.mux.Lock()
	defer bus.mux.Unlock()

	bus.handlers[channel] = append(bus.handlers[channel], fn)
}

func (bus *EventBus) Broadcast(channel string, data interface{}) {
	bus.bus.Publish(channel, data)
}

func (bus *EventBus) Subscribe(channel string, cb Callback) {
	_ = bus.bus.SubscribeAsync(channel, cb, false)
}

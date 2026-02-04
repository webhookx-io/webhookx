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
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.uber.org/zap"
)

const channelName = "webhookx"

type PostgresEventBus struct {
	ctx    context.Context
	cancel context.CancelFunc

	nodeID   string
	log      *zap.SugaredLogger
	listener *pq.Listener
	mux      sync.Mutex
	handlers map[string][]ClusteringHandler
	bus      evbus.Bus
	db       *sql.DB
}

func NewPostgresEventBus(nodeID string, dsn string, log *zap.SugaredLogger, db *sql.DB) *PostgresEventBus {
	ctx, cancel := context.WithCancel(context.Background())
	bus := PostgresEventBus{
		ctx:      ctx,
		cancel:   cancel,
		bus:      evbus.New(),
		nodeID:   nodeID,
		listener: pq.NewListener(dsn, time.Millisecond*100, time.Minute, nil),
		handlers: make(map[string][]ClusteringHandler),
		log:      log,
		db:       db,
	}

	return &bus
}

func (b *PostgresEventBus) Start() error {
	go b.listenClusterLoop()
	go b.startListen()
	return nil
}

func (b *PostgresEventBus) startListen() {
	err := b.listener.Listen(channelName)
	if err != nil {
		b.log.Errorf("failed to listen on channel %s: %v", channelName, err)
		return
	}
	b.log.Infof(`listening on channel "%s"`, channelName)
}

func (b *PostgresEventBus) Name() string {
	return "eventbus"
}

func (b *PostgresEventBus) Stop(ctx context.Context) error {
	b.cancel()
	_ = b.listener.Close()
	return nil
}

func (b *PostgresEventBus) shouldHandle(e *postgresEvent) bool {
	return b.nodeID != e.Node
}

func (b *PostgresEventBus) callClusteringHandlers(e *postgresEvent) {
	b.mux.Lock()
	defer b.mux.Unlock()

	if handlers, ok := b.handlers[e.Event]; ok {
		for _, handler := range handlers {
			handler(e.Data)
		}
	}
}

func (b *PostgresEventBus) listenClusterLoop() {
	timeoutDuration := 5 * time.Second
	timeout := time.NewTimer(timeoutDuration)
	defer timeout.Stop()

	for {
		timeout.Reset(timeoutDuration)
		select {
		case <-b.ctx.Done():
			return
		case n := <-b.listener.NotificationChannel():
			var event postgresEvent
			if err := json.Unmarshal([]byte(n.Extra), &event); err != nil {
				b.log.Warnf("failed to marshal postgres event: %s", err)
				continue
			}
			if !b.shouldHandle(&event) {
				continue
			}
			b.log.Debugf("calling handlers: %s", n.Extra)
			b.callClusteringHandlers(&event)
		case <-timeout.C:
			err := b.listener.Ping()
			if err != nil {
				b.log.Errorf("failed to ping database: %v", err)
			}
		}
	}
}

func (b *PostgresEventBus) ClusteringBroadcast(ctx context.Context, channel string, value Marshaler) error {
	ctx, span := tracing.Start(ctx, "bus.clustering_broadcast")
	defer span.End()

	b.Broadcast(ctx, channel, value)

	bytes, err := value.Marshal()
	if err != nil {
		b.log.Errorf("failed to marshal value: %v", err)
		return err
	}
	event := postgresEvent{
		Event: channel,
		Time:  time.Now().UnixMilli(),
		Node:  b.nodeID,
		Data:  bytes,
	}
	bytes, err = json.Marshal(event)
	if err != nil {
		b.log.Errorf("failed to marshal postgres event: %v", err)
		return err
	}

	b.log.Debugf("broadcasting cluster event: %s", string(bytes))

	statement := fmt.Sprintf("NOTIFY %s, %s", channelName, pq.QuoteLiteral(string(bytes)))
	_, err = b.db.ExecContext(ctx, statement)
	if err != nil {
		b.log.Errorf("broadcasting cluster event error: %v", err)
	}
	return err
}

func (b *PostgresEventBus) ClusteringSubscribe(channel string, handler ClusteringHandler) {
	b.mux.Lock()
	defer b.mux.Unlock()

	b.handlers[channel] = append(b.handlers[channel], handler)
}

func (b *PostgresEventBus) Broadcast(ctx context.Context, channel string, value interface{}) {
	_, span := tracing.Start(ctx, "bus.broadcast")
	defer span.End()

	b.bus.Publish(channel, value)
}

func (b *PostgresEventBus) Subscribe(channel string, handler Handler) {
	_ = b.bus.SubscribeAsync(channel, handler, false)
}

type postgresEvent struct {
	Event string          `json:"event"`
	Time  int64           `json:"time"`
	Node  string          `json:"node"`
	Data  json.RawMessage `json:"data"`
}

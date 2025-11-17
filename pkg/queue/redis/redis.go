package redis

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/pkg/loglimiter"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type RedisQueue struct {
	opts    Options
	c       *redis.Client
	log     *zap.SugaredLogger
	limiter *loglimiter.Limiter
}

type Options struct {
	StreamName        string
	ConsumerGroupName string
	ConsumerName      string
	VisibilityTimeout time.Duration
	Listeners         int

	Client *redis.Client
}

func NewRedisQueue(opts Options, logger *zap.SugaredLogger) (queue.Queue, error) {
	q := &RedisQueue{
		opts:    opts,
		c:       opts.Client,
		log:     logger.Named("queue-redis"),
		limiter: loglimiter.NewLimiter(time.Second),
	}
	return q, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, message *queue.Message) error {
	ctx, span := tracing.Start(ctx, "redis.queue.enqueue", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	args := &redis.XAddArgs{
		Stream: q.opts.StreamName,
		ID:     "*",
		Values: []interface{}{
			"data", message.Value,
			"time", message.Time.UnixMilli(),
			"ws_id", message.WorkspaceID,
		},
	}
	return q.c.XAdd(ctx, args).Err()
}

func toMessage(values map[string]interface{}) *queue.Message {
	message := &queue.Message{}

	if data, ok := values["data"].(string); ok {
		message.Value = []byte(data)
	}

	if timestr, ok := values["time"].(string); ok {
		t, _ := strconv.ParseInt(timestr, 10, 64)
		message.Time = time.UnixMilli(t)
	}

	if wsid, ok := values["ws_id"].(string); ok {
		message.WorkspaceID = wsid
	}

	return message
}

func (q *RedisQueue) dequeue(ctx context.Context) ([]redis.XMessage, error) {
	ctx, span := tracing.Start(ctx, "redis.queue.dequeue", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	args := &redis.XReadGroupArgs{
		Group:    q.opts.ConsumerGroupName,
		Consumer: q.opts.ConsumerName,
		Streams:  []string{q.opts.StreamName, ">"},
		Count:    20,
		Block:    time.Second,
	}
	streams, err := q.c.XReadGroup(ctx, args).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = nil
		} else if strings.HasPrefix(err.Error(), "NOGROUP") {
			err = nil
			go q.createConsumerGroup(q.opts.StreamName, q.opts.ConsumerGroupName)
		}
		return nil, err
	}

	return streams[0].Messages, nil
}

func (q *RedisQueue) delete(ctx context.Context, xmessages []redis.XMessage) error {
	ctx, span := tracing.Start(ctx, "redis.queue.delete", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ids := make([]string, 0, len(xmessages))
	for _, message := range xmessages {
		ids = append(ids, message.ID)
	}

	pipeline := q.c.Pipeline()
	pipeline.XAck(ctx, q.opts.StreamName, q.opts.ConsumerGroupName, ids...)
	pipeline.XDel(ctx, q.opts.StreamName, ids...)
	_, err := pipeline.Exec(ctx)
	return err
}

func (q *RedisQueue) StartListen(ctx context.Context, handler queue.HandlerFunc) {
	q.log.Infof("starting %d listeners", q.opts.Listeners)
	for i := 0; i < q.opts.Listeners; i++ {
		go q.listen(ctx, handler)
	}
	go q.process(ctx)
}

func (q *RedisQueue) listen(ctx context.Context, handler queue.HandlerFunc) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			xmessages, err := q.dequeue(ctx)
			if err != nil && q.limiter.Allow(err.Error()) {
				q.log.Warnf("failed to dequeue: %v", err)
				time.Sleep(time.Second)
				continue
			}
			if len(xmessages) == 0 {
				continue
			}

			messages := make([]*queue.Message, 0, len(xmessages))
			for _, msg := range xmessages {
				messages = append(messages, toMessage(msg.Values))
			}

			err = handler(ctx, messages)
			if err != nil {
				q.log.Warnf("failed to handle message: %v", err)
				continue
			}
			err = q.delete(ctx, xmessages)
			if err != nil {
				q.log.Warnf("failed to delete message: %v", err)
			}
		}
	}
}

func (q *RedisQueue) size(ctx context.Context) (int64, error) {
	return q.c.XLen(ctx, q.opts.StreamName).Result()
}

func (q *RedisQueue) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	size, err := q.size(context.TODO())
	if err != nil {
		q.log.Errorf("failed to retrieve status: %v", err)
	}
	stats["eventqueue.size"] = size

	return stats
}

func (q *RedisQueue) createConsumerGroup(stream string, group string) {
	err := q.c.XGroupCreateMkStream(context.TODO(), stream, group, "0").Err()
	if err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			q.log.Errorf("failed to create Consumer Group '%s': %s", group, err.Error())
		}
		return
	}
	q.log.Debugf("Consumer Group '%s' created", group)
}

func (q *RedisQueue) process(ctx context.Context) {
	var reenqueueScript = redis.NewScript(`
		local entries = redis.call('XPENDING', KEYS[1], KEYS[2], 'IDLE', ARGV[1], '-', '+', 1000)
		local ids = {}
		if entries then 
			for i, entry in ipairs(entries) do
				local id = entry[1]
				local res = redis.call('XRANGE', KEYS[1], id, id)
				local items = res[1][2]
				local new_id = redis.call('XADD', KEYS[1], '*', unpack(items))
				ids[i] = new_id
				redis.call('XACK', KEYS[1], KEYS[2], id)
				redis.call('XDEL', KEYS[1], id)
			end
		end
		return ids
	`)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			keys := []string{q.opts.StreamName, q.opts.ConsumerGroupName}
			argv := []interface{}{q.opts.VisibilityTimeout.Milliseconds()}
			res, err := reenqueueScript.Run(context.TODO(), q.c, keys, argv...).Result()
			if err != nil {
				q.log.Errorf("failed to reenqueue: %v", err)
				continue
			}

			if ids, ok := res.([]interface{}); ok && len(ids) > 0 {
				q.log.Debugf("enqueued invisible messages: %d", len(ids))
			}
		}
	}
}

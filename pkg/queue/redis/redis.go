package redis

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

type RedisQueue struct {
	stream            string
	group             string
	consumer          string
	visibilityTimeout time.Duration

	c       *redis.Client
	log     *zap.SugaredLogger
	metrics *metrics.Metrics
}

type RedisQueueOptions struct {
	StreamName        string
	GroupName         string
	ConsumerName      string
	VisibilityTimeout time.Duration

	Client *redis.Client
}

func NewRedisQueue(opts RedisQueueOptions, logger *zap.SugaredLogger, metrics *metrics.Metrics) (queue.Queue, error) {
	q := &RedisQueue{
		stream:            utils.DefaultIfZero(opts.StreamName, constants.QueueRedisQueueName),
		group:             utils.DefaultIfZero(opts.GroupName, constants.QueueRedisGroupName),
		consumer:          utils.DefaultIfZero(opts.ConsumerName, constants.QueueRedisConsumerName),
		visibilityTimeout: utils.DefaultIfZero(opts.VisibilityTimeout, constants.QueueRedisVisibilityTimeout),
		c:                 opts.Client,
		log:               logger,
		metrics:           metrics,
	}

	go q.process()
	if metrics.Enabled {
		go q.monitoring()
	}

	return q, nil
}

func (q *RedisQueue) Enqueue(ctx context.Context, message *queue.Message) error {
	ctx, span := tracing.Start(ctx, "redis.queue.enqueue", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	args := &redis.XAddArgs{
		Stream: q.stream,
		ID:     "*",
		Values: []interface{}{"data", message.Data, "time", message.Time.UnixMilli(), "ws_id", message.WorkspaceID},
	}
	res := q.c.XAdd(ctx, args)
	if res.Err() != nil {
		return res.Err()
	}
	message.ID = res.Val()
	return nil
}

func toMessage(values map[string]interface{}) *queue.Message {
	message := &queue.Message{}

	if data, ok := values["data"].(string); ok {
		message.Data = []byte(data)
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

func (q *RedisQueue) Dequeue(ctx context.Context, opt *queue.Options) ([]*queue.Message, error) {
	ctx, span := tracing.Start(ctx, "redis.queue.dequeue", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	var count int64 = 1
	if opt != nil && opt.Count != 0 {
		count = opt.Count
	}
	var block time.Duration = -1
	if opt != nil && opt.Block {
		block = opt.Timeout
	}

	args := &redis.XReadGroupArgs{
		Group:    q.group,
		Consumer: q.consumer,
		Streams:  []string{q.stream, ">"},
		Count:    count,
		Block:    block,
	}
	res := q.c.XReadGroup(ctx, args)
	if res.Err() != nil {
		err := res.Err()
		if errors.Is(err, redis.Nil) {
			err = nil
		} else if strings.HasPrefix(err.Error(), "NOGROUP") {
			go q.createConsumerGroup()
			err = nil
		}
		return nil, err
	}

	messages := make([]*queue.Message, 0)
	for _, stream := range res.Val() {
		for _, xmessage := range stream.Messages {
			message := toMessage(xmessage.Values)
			message.ID = xmessage.ID
			messages = append(messages, message)
		}
	}

	return messages, nil
}

func (q *RedisQueue) Delete(ctx context.Context, messages []*queue.Message) error {
	ctx, span := tracing.Start(ctx, "redis.queue.delete", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ids := make([]string, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}
	pipeline := q.c.Pipeline()
	pipeline.XAck(ctx, q.stream, q.group, ids...)
	pipeline.XDel(ctx, q.stream, ids...)
	_, err := pipeline.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (q *RedisQueue) Size(ctx context.Context) (int64, error) {
	return q.c.XLen(ctx, q.stream).Result()
}

func (q *RedisQueue) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	size, err := q.Size(context.TODO())
	if err != nil {
		q.log.Errorf("failed to retrieve status: %v", err)
	}
	stats["eventqueue.size"] = size

	return stats
}

func (q *RedisQueue) createConsumerGroup() {
	res := q.c.XGroupCreateMkStream(context.TODO(), q.stream, q.group, "0")
	if res.Err() == nil {
		q.log.Debugf("created default consumer group: %s", q.group)
		return
	}

	if res.Err().Error() != "BUSYGROUP Consumer Group name already exists" {
		q.log.Errorf("failed to create the default consumer group: %s", res.Err().Error())
	}
}

// process re-enqueue invisible messages that reach the visibility timeout
func (q *RedisQueue) process() {
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

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				keys := []string{q.stream, q.group}
				argv := []interface{}{q.visibilityTimeout.Milliseconds()}
				res, err := reenqueueScript.Run(context.Background(), q.c, keys, argv...).Result()
				if err != nil {
					q.log.Errorf("failed to reenqueue: %v", err)
					continue
				}

				if ids, ok := res.([]interface{}); ok && len(ids) > 0 {
					q.log.Debugf("enqueued invisible messages: %d", len(ids))
				}
			}
		}

	}()
}

func (q *RedisQueue) monitoring() {
	ticker := time.NewTicker(q.metrics.Interval)
	defer ticker.Stop()
	for range ticker.C {
		size, err := q.Size(context.TODO())
		if err != nil {
			q.log.Errorf("failed to get redis queue size: %v", err)
			continue
		}
		q.metrics.EventPendingGauge.Set(float64(size))
	}
}

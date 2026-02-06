package taskqueue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

var (
	getMultiScript = redis.NewScript(`
		redis.replicate_commands()
		local key_queue = KEYS[1]
		local key_queue_data = KEYS[2]
		local time = redis.call('TIME')
		local now = time[1] * 1000 + math.floor(time[2] / 1000)
		local timeout = now + ARGV[2]
		local res = redis.call('ZRANGE', key_queue, 0, now, 'BYSCORE', 'LIMIT', 0, ARGV[1], 'WITHSCORES')
		local n = 1
		local list = {}
		for i = 1, #res, 2 do
			local id = res[i]
			local score = tonumber(res[i + 1])
			local data = redis.call('HGET', key_queue_data, id)
			redis.call('ZADD', key_queue, timeout, id)
			list[n] = { id, score, data }
			n = n + 1
		end
		
		return list
	`)
)

// RedisTaskQueue use redis as queue implementation
type RedisTaskQueue struct {
	queue             string
	queueData         string
	visibilityTimeout time.Duration

	c       *redis.Client
	log     *zap.SugaredLogger
	metrics *metrics.Metrics
}

type RedisTaskQueueOptions struct {
	QueueName         string
	QueueDataName     string
	VisibilityTimeout time.Duration
	Client            *redis.Client
}

func NewRedisQueue(opts RedisTaskQueueOptions, logger *zap.SugaredLogger, metrics *metrics.Metrics) *RedisTaskQueue {
	q := &RedisTaskQueue{
		queue:             utils.DefaultIfZero(opts.QueueName, constants.TaskQueueName),
		visibilityTimeout: utils.DefaultIfZero(opts.VisibilityTimeout, constants.TaskQueueVisibilityTimeout),
		queueData:         utils.DefaultIfZero(opts.QueueDataName, constants.TaskQueueDataName),
		c:                 opts.Client,
		log:               logger.Named("queue.task"),
		metrics:           metrics,
	}

	if metrics != nil && metrics.Enabled {
		go q.monitoring()
	}

	return q
}

func (q *RedisTaskQueue) Add(ctx context.Context, tasks []*TaskMessage) error {
	ctx, span := tracing.Start(ctx, "task_queue.redis.add")
	defer span.End()

	// TODO: inject trace context

	members := make([]redis.Z, len(tasks))
	strs := make([]interface{}, 0, len(tasks)*2)
	ids := make([]string, len(tasks))
	for i, task := range tasks {
		members[i] = redis.Z{
			Score:  float64(task.ScheduledAt.UnixMilli()),
			Member: task.ID,
		}
		data, err := task.MarshalData()
		if err != nil {
			return err
		}
		strs = append(strs, task.ID, data)
		ids[i] = task.ID
	}
	q.log.Debugw("adding tasks", "tasks", ids)
	pipeline := q.c.Pipeline()
	pipeline.HSet(ctx, q.queueData, strs...)
	pipeline.ZAdd(ctx, q.queue, members...)
	_, err := pipeline.Exec(ctx)
	return err
}

func (q *RedisTaskQueue) Schedule(ctx context.Context, id string, scheduledAt time.Time) error {
	ctx, span := tracing.Start(ctx, "task_queue.redis.schedule")
	span.SetAttributes(attribute.String("id", id))
	span.SetAttributes(attribute.Int64("timestamp", scheduledAt.UnixMilli()))
	defer span.End()

	q.log.Debugf("scheduling task %s at %s", id, scheduledAt)
	return q.c.ZAdd(ctx, q.queue, redis.Z{
		Score:  float64(scheduledAt.UnixMilli()),
		Member: id,
	}).Err()
}

func decode(parts []interface{}) *TaskMessage {
	// TODO: extract trace context
	task := &TaskMessage{}
	task.ID = parts[0].(string)
	task.ScheduledAt = time.UnixMilli(parts[1].(int64))
	if len(parts) >= 3 && parts[2] != nil {
		task.data = []byte((parts[2].(string)))
	}
	return task
}

func (q *RedisTaskQueue) Get(ctx context.Context, opts *GetOptions) ([]*TaskMessage, error) {
	ctx, span := tracing.Start(ctx, "task_queue.redis.get")
	defer span.End()

	keys := []string{q.queue, q.queueData}
	argv := []interface{}{
		opts.Count,
		q.visibilityTimeout.Milliseconds(),
	}
	res, err := getMultiScript.Run(ctx, q.c, keys, argv...).Result()
	if err != nil {
		return nil, err
	}
	switch list := res.(type) {
	case []interface{}:
		if len(list) == 0 {
			return nil, nil
		}
		tasks := make([]*TaskMessage, len(list))
		for i, v := range list {
			tasks[i] = decode(v.([]interface{}))
		}
		return tasks, nil
	default:
		return nil, fmt.Errorf("unexpected return value: expect array, got %s", res)
	}
}

func (q *RedisTaskQueue) Delete(ctx context.Context, ids ...string) error {
	ctx, span := tracing.Start(ctx, "task_queue.redis.delete")
	span.SetAttributes(attribute.StringSlice("id", ids))
	defer span.End()

	q.log.Debugw("deleting task", "ids", ids)

	pipeline := q.c.Pipeline()
	pipeline.HDel(ctx, q.queueData, ids...)
	pipeline.ZRem(ctx, q.queue, ids)
	_, err := pipeline.Exec(ctx)
	return err
}

func (q *RedisTaskQueue) Size(ctx context.Context) (int64, error) {
	return q.c.ZCard(ctx, q.queue).Result()
}

func (q *RedisTaskQueue) Stats() map[string]interface{} {
	stats := make(map[string]interface{})

	size, err := q.Size(context.TODO())
	if err != nil {
		q.log.Errorf("failed to retrieve size: %v", err)
	}
	stats["queue.size"] = size

	now := time.Now()
	res, err := q.c.ZRangeByScoreWithScores(context.TODO(), constants.TaskQueueName, &redis.ZRangeBy{
		Min:    "0",
		Max:    strconv.FormatInt(now.UnixMilli(), 10),
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		q.log.Errorf("failed to retrieve backlog_latency: %v", err)
	}

	if len(res) > 0 {
		seconds := (now.UnixMilli() - int64(res[0].Score)) / 1000
		stats["queue.backlog_latency"] = seconds
	}

	return stats
}

func (q *RedisTaskQueue) monitoring() {
	ticker := time.NewTicker(q.metrics.Interval)
	defer ticker.Stop()
	for range ticker.C {
		size, err := q.Size(context.TODO())
		if err != nil {
			q.log.Errorf("failed to get task queue size: %v", err)
			continue
		}
		q.metrics.AttemptPendingGauge.Set(float64(size))
	}
}

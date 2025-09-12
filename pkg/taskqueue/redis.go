package taskqueue

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var (
	getMultiScript = redis.NewScript(`
		redis.replicate_commands()
		local key_queue = KEYS[1]
		local key_queue_data = KEYS[2]
		local time = redis.call('TIME')
		local now = time[1] * 1000 + math.floor(time[2] / 1000)
		local task_ids = redis.call('ZRANGEBYSCORE', key_queue, 0, now, 'LIMIT', 0, ARGV[1])
		local list = {}
	
		if task_ids and task_ids[1] then
			local timeout = now + ARGV[2]
			for i, task_id in ipairs(task_ids) do
				local data = redis.call('HGET', key_queue_data, task_id)
				if not data then
					redis.call("ZREM", key_queue, task_id)
				end
				redis.call('ZADD', key_queue, timeout, task_id)
				list[i] = { task_id, data }
			end
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
	ctx, span := tracing.Start(ctx, "taskqueue.redis.add", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	members := make([]redis.Z, 0, len(tasks))
	strs := make([]interface{}, 0, len(tasks)*2)
	ids := make([]string, 0, len(tasks))
	for _, task := range tasks {
		members = append(members, redis.Z{
			Score:  float64(task.ScheduledAt.UnixMilli()),
			Member: task.ID,
		})
		data, err := task.MarshalData()
		if err != nil {
			return err
		}
		strs = append(strs, task.ID, data)
		ids = append(ids, task.ID)
	}
	q.log.Debugw("adding tasks", "tasks", ids)
	pipeline := q.c.Pipeline()
	pipeline.HSet(ctx, q.queueData, strs...)
	pipeline.ZAdd(ctx, q.queue, members...)
	_, err := pipeline.Exec(ctx)
	return err
}

func (q *RedisTaskQueue) Schedule(ctx context.Context, task *TaskMessage) error {
	q.log.Debugf("scheduling task %s at %s", task.ID, task.ScheduledAt)
	return q.c.ZAdd(ctx, q.queue, redis.Z{
		Score:  float64(task.ScheduledAt.UnixMilli()),
		Member: task.ID,
	}).Err()
}

func (q *RedisTaskQueue) Get(ctx context.Context, opts *GetOptions) ([]*TaskMessage, error) {
	ctx, span := tracing.Start(ctx, "taskqueue.redis.get", trace.WithSpanKind(trace.SpanKindServer))
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
		tasks := make([]*TaskMessage, 0, len(list))
		for _, e := range list {
			array := e.([]interface{})
			if len(array) == 2 {
				tasks = append(tasks, &TaskMessage{
					ID:   array[0].(string),
					data: []byte((array[1].(string))),
				})
			}
		}
		return tasks, nil
	default:
		return nil, fmt.Errorf("unexpected return value: expect array, got %s", res)
	}
}

func (q *RedisTaskQueue) Delete(ctx context.Context, task *TaskMessage) error {
	ctx, span := tracing.Start(ctx, "taskqueue.redis.delete", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	q.log.Debugf("deleting task %s", task.ID)
	pipeline := q.c.Pipeline()
	pipeline.HDel(ctx, q.queueData, task.ID)
	pipeline.ZRem(ctx, q.queue, task.ID)
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

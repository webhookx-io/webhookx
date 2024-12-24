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
	"time"
)

var (
	getMultiScript = redis.NewScript(`
		redis.replicate_commands()
		local time = redis.call('TIME')
		local now = time[1] * 1000 + math.floor(time[2] / 1000)
		local task_ids = redis.call('ZRANGEBYSCORE', KEYS[1], 0, now, 'LIMIT', 0, ARGV[1])
		local list = {}
		if task_ids and task_ids[1] then
			for i, task_id in ipairs(task_ids) do
				local data = redis.call('HGET', KEYS[2], task_id)
				if not data then
					redis.call("ZREM", KEYS[1], task_id)
					redis.call("ZREM", KEYS[3], task_id)
				else
					redis.call("ZREM", KEYS[1], task_id)
					redis.call('ZADD', KEYS[3], now + ARGV[2], task_id)
					list[i] = { task_id, data }
				end
			end
		end
		
		return list
	`)

	requeueScript = redis.NewScript(`
		redis.replicate_commands()
		local now = redis.call('TIME')[1]
		local tasks = redis.call('ZRANGEBYSCORE', KEYS[1], 0, now)
		local n = 0
		if tasks then
			for _, id in ipairs(tasks) do
				n = n + 1
				redis.call("ZREM", KEYS[1], id)
				redis.call('ZADD', KEYS[2], now, id)
			end
		end
		return tasks
	`)
)

// RedisTaskQueue use redis as queue implementation
type RedisTaskQueue struct {
	queue             string
	invisibleQueue    string
	queueData         string
	visibilityTimeout time.Duration

	c       *redis.Client
	log     *zap.SugaredLogger
	metrics *metrics.Metrics
}

type RedisTaskQueueOptions struct {
	QueueName          string
	InvisibleQueueName string
	QueueDataName      string
	VisibilityTimeout  time.Duration
	Client             *redis.Client
}

func NewRedisQueue(opts RedisTaskQueueOptions, logger *zap.SugaredLogger, metrics *metrics.Metrics) *RedisTaskQueue {
	q := &RedisTaskQueue{
		queue:             utils.DefaultIfZero(opts.QueueName, constants.TaskQueueName),
		invisibleQueue:    utils.DefaultIfZero(opts.InvisibleQueueName, constants.TaskQueueInvisibleQueueName),
		visibilityTimeout: utils.DefaultIfZero(opts.VisibilityTimeout, constants.TaskQueueVisibilityTimeout),
		queueData:         utils.DefaultIfZero(opts.QueueDataName, constants.TaskQueueDataName),
		c:                 opts.Client,
		log:               logger,
		metrics:           metrics,
	}
	q.process()

	if metrics.Enabled {
		go q.monitoring()
	}

	return q
}

func (q *RedisTaskQueue) Add(ctx context.Context, tasks []*TaskMessage) error {
	ctx, span := tracing.Start(ctx, "taskqueue.redis.add", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	members := make([]redis.Z, 0, len(tasks))
	strs := make([]interface{}, 0, len(tasks)*2)
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
	}

	pipeline := q.c.Pipeline()
	pipeline.HSet(ctx, q.queueData, strs...)
	pipeline.ZAdd(ctx, q.queue, members...)
	_, err := pipeline.Exec(ctx)
	return err
}

func (q *RedisTaskQueue) Get(ctx context.Context, opts *GetOptions) ([]*TaskMessage, error) {
	ctx, span := tracing.Start(ctx, "taskqueue.redis.get", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	keys := []string{q.queue, q.queueData, q.invisibleQueue}
	argv := []interface{}{
		opts.Count,
		q.visibilityTimeout.Milliseconds() / 1000,
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
			task := e.([]interface{})
			tasks = append(tasks, &TaskMessage{
				ID:   task[0].(string),
				data: []byte((task[1].(string))),
			})
		}
		return tasks, nil
	default:
		return nil, fmt.Errorf("[redis-queue] unexpected return value: expect array, got %s", res)
	}
}

func (q *RedisTaskQueue) Delete(ctx context.Context, task *TaskMessage) error {
	ctx, span := tracing.Start(ctx, "taskqueue.redis.delete", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	q.log.Debugf("[redis-queue]: delete task %s", task.ID)
	pipeline := q.c.Pipeline()
	pipeline.HDel(ctx, q.queueData, task.ID)
	pipeline.ZRem(ctx, q.invisibleQueue, task.ID)
	pipeline.ZRem(ctx, q.queue, task.ID)
	_, err := pipeline.Exec(ctx)
	return err
}

func (q *RedisTaskQueue) Size(ctx context.Context) (int64, error) {
	return q.c.ZCard(ctx, q.queue).Result()
}

// process re-enqueue invisible tasks that reach the visibility timeout
func (q *RedisTaskQueue) process() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				keys := []string{q.invisibleQueue, q.queue}
				res, err := requeueScript.Run(context.Background(), q.c, keys).Result()
				if err != nil {
					q.log.Errorf("failed to run requeue script: %s", err)
					continue
				}
				if ids, ok := res.([]interface{}); ok && len(ids) > 0 {
					q.log.Debugf("enqueued invisible tasks: %v", ids)
				}
			}
		}
	}()
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

package taskqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"time"
)

var (
	addScript = redis.NewScript(`
		local score = ARGV[1]
		local task_id = ARGV[2]
		local queue_data = ARGV[3]
		redis.call('ZADD', KEYS[1], score, task_id)
		redis.call('HSET', KEYS[2], task_id, queue_data)
		return 1
	`)

	getScript = redis.NewScript(`
		redis.replicate_commands()
		local time = redis.call('TIME')
		local now = time[1] * 1000 + math.floor(time[2] / 1000)
		local keys = redis.call('ZRANGEBYSCORE', KEYS[1], 0, now, 'LIMIT', 0, 1)
		local task_id = keys and keys[1]
		local invisible_timeout = ARGV[1]
		if task_id then
			redis.call("ZREM", KEYS[1], task_id)
			redis.call('ZADD', KEYS[3], now + invisible_timeout, task_id)
			local queue_data = redis.call('HGET', KEYS[2], task_id)
			return { task_id, queue_data }
		end
		return {}
	`)

	deleteScript = redis.NewScript(`
		redis.replicate_commands()
		local task_id = ARGV[1]
		redis.call("ZREM", KEYS[1], task_id)
		redis.call("ZREM", KEYS[2], task_id)
		redis.call("HDEL", KEYS[3], task_id)
		return 1
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

	c   *redis.Client
	log *zap.SugaredLogger
}

type RedisTaskQueueOptions struct {
	QueueName          string
	InvisibleQueueName string
	QueueDataName      string
	VisibilityTimeout  time.Duration
	Client             *redis.Client
}

func NewRedisQueue(opts RedisTaskQueueOptions, logger *zap.SugaredLogger) *RedisTaskQueue {
	q := &RedisTaskQueue{
		queue:             utils.DefaultIfZero(opts.QueueName, constants.TaskQueueName),
		invisibleQueue:    utils.DefaultIfZero(opts.InvisibleQueueName, constants.TaskQueueInvisibleQueueName),
		visibilityTimeout: utils.DefaultIfZero(opts.VisibilityTimeout, constants.TaskQueueVisibilityTimeout),
		queueData:         utils.DefaultIfZero(opts.QueueDataName, constants.TaskQueueDataName),
		c:                 opts.Client,
		log:               logger,
	}
	q.process()
	return q
}

func (q *RedisTaskQueue) Add(ctx context.Context, task *TaskMessage, scheduleAt time.Time) error {
	q.log.Debugf("[redis-queue]: add task %s schedule at: %s", task.ID, scheduleAt.Format("2006-01-02T15:04:05.000"))
	keys := []string{q.queue, q.queueData}
	data, err := json.Marshal(task.Data)
	if err != nil {
		return err
	}
	argv := []interface{}{
		scheduleAt.UnixMilli(),
		task.ID,
		data,
	}
	res, err := addScript.Run(ctx, q.c, keys, argv...).Result()
	if err != nil {
		return err
	}
	if v, ok := res.(int64); !ok || v != 1 {
		return fmt.Errorf("[redis-queue] unexpected return value: expect 1, got %v", v)
	}
	return nil
}

func (q *RedisTaskQueue) Get(ctx context.Context) (*TaskMessage, error) {
	keys := []string{q.queue, q.queueData, q.invisibleQueue}
	argv := []interface{}{
		q.visibilityTimeout.Milliseconds() / 1000,
	}
	res, err := getScript.Run(ctx, q.c, keys, argv...).Result()
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case []interface{}:
		if len(v) == 0 {
			return nil, nil
		}

		var task TaskMessage
		task.ID = v[0].(string)
		task.data = []byte((v[1].(string)))
		return &task, nil
	default:
		return nil, fmt.Errorf("[redis-queue] unexpected return value: expect array, got %s", res)
	}
}

func (q *RedisTaskQueue) Delete(ctx context.Context, task *TaskMessage) error {
	q.log.Debugf("[redis-queue]: delete task %s", task.ID)
	keys := []string{q.invisibleQueue, q.queue, q.queueData}
	argv := []interface{}{
		task.ID,
	}
	res, err := deleteScript.Run(ctx, q.c, keys, argv...).Result()
	if err != nil {
		return err
	}
	if v, ok := res.(int64); !ok || v != 1 {
		return fmt.Errorf("[redis-queue] unexpected return value: expect 1, got %v", v)
	}
	return nil
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

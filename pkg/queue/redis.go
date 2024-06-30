package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"time"
)

const (
	DefaultQueueName   = "webhookx:queue"
	InvisibleQueueName = "webhookx:queue_invisible"
	QueueDataHashName  = "webhookx:queue_data"
	VisibilityTimeout  = 60
)

var (
	addScript = redis.NewScript(`
		redis.replicate_commands()
		local now = redis.call('TIME')[1]
		local score = now + tonumber(ARGV[1])
		local task_id = ARGV[2]
		local queue_data = ARGV[3]
		redis.call('ZADD', KEYS[1], score, task_id)
		redis.call('HSET', KEYS[2], task_id, queue_data)
		return 1
	`)

	getScript = redis.NewScript(`
		redis.replicate_commands()
		local now = redis.call('TIME')[1]
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

	reenqueueScript = redis.NewScript(`
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

// RedisTaskQueue use redis as queue implemention
type RedisTaskQueue struct {
	queue string
	c     *redis.Client
}

func NewRedisQueue(client *redis.Client) *RedisTaskQueue {
	q := &RedisTaskQueue{
		c:     client,
		queue: "webhookx:queue",
	}
	q.process()
	return q
}

func (q *RedisTaskQueue) Add(task *Task, delay time.Duration) error {
	keys := []string{DefaultQueueName, QueueDataHashName}
	data, err := json.Marshal(task.Data)
	if err != nil {
		return err
	}
	argv := []interface{}{
		delay.Milliseconds() / 1000,
		task.ID,
		data,
	}
	res, err := addScript.Run(context.Background(), q.c, keys, argv...).Result()
	if err != nil {
		return err
	}
	if v, ok := res.(int64); !ok || v != 1 {
		return fmt.Errorf("[redis-queue] unexpected return value: expect 1, got %v", v)
	}
	return nil
}

func (q *RedisTaskQueue) Get() (*Task, error) {
	keys := []string{DefaultQueueName, QueueDataHashName, InvisibleQueueName}
	argv := []interface{}{
		VisibilityTimeout,
	}
	res, err := getScript.Run(context.Background(), q.c, keys, argv...).Result()
	if err != nil {
		return nil, err
	}
	switch v := res.(type) {
	case []interface{}:
		if len(v) == 0 {
			return nil, nil
		}

		var task Task
		task.ID = v[0].(string)
		task.data = []byte((v[1].(string)))
		return &task, nil
	default:
		return nil, fmt.Errorf("[redis-queue] unexpected return value: expect array, got %s", res)
	}
}

func (q *RedisTaskQueue) Delete(task *Task) error {
	keys := []string{InvisibleQueueName, DefaultQueueName, QueueDataHashName}
	argv := []interface{}{
		task.ID,
	}
	res, err := deleteScript.Run(context.Background(), q.c, keys, argv...).Result()
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
				keys := []string{InvisibleQueueName, DefaultQueueName}
				res, err := reenqueueScript.Run(context.Background(), q.c, keys).Result()
				if err != nil {
					zap.S().Errorf("failed to : %s", err)
					continue
				}
				if ids, ok := res.([]interface{}); ok && len(ids) > 0 {
					zap.S().Debugf("enqueued invisible tasks: %v", ids)
				}
			}
		}
	}()
}

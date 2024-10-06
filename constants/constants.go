package constants

import (
	"github.com/webhookx-io/webhookx/config"
	"time"
)

// Task Queue
const (
	TaskQueueName               = "webhookx:queue"
	TaskQueueInvisibleQueueName = "webhookx:queue_invisible"
	TaskQueueDataName           = "webhookx:queue_data"
	TaskQueueVisibilityTimeout  = time.Second * 60
)

// Redis Queue
const (
	QueueRedisQueueName         = "webhookx:proxy_queue"
	QueueRedisGroupName         = "group_default"
	QueueRedisConsumerName      = "consumer_default"
	QueueRedisVisibilityTimeout = time.Second * 60
)

const (
	RequeueBatch    = 100
	RequeueInterval = time.Second * 60
)

var (
	DefaultResponseHeaders = map[string]string{
		"Content-Type": "application/json",
		"Server":       "WebhookX/" + config.VERSION,
	}
	DefaultDelivererRequestHeaders = map[string]string{
		"User-Agent":   "WebhookX/" + config.VERSION,
		"Content-Type": "application/json; charset=utf-8",
	}
)

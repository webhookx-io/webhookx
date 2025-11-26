package constants

import (
	"time"

	"github.com/webhookx-io/webhookx/config"
)

// Task Queue
const (
	TaskQueueName                  = "webhookx:queue"
	TaskQueueDataName              = "webhookx:queue_data"
	TaskQueueVisibilityTimeout     = time.Second * 65
	TaskQueuePreScheduleTimeWindow = time.Minute * 3
)

// Redis Queue
const (
	QueueRedisQueueName         = "webhookx:proxy_queue"
	QueueRedisGroupName         = "group_default"
	QueueRedisConsumerName      = "consumer_default"
	QueueRedisVisibilityTimeout = time.Second * 60
)

type Header struct {
	Name  string
	Value string
}

var (
	HeaderEventId          = "X-Webhookx-Event-Id"
	DefaultResponseHeaders = []Header{
		{Name: "Server", Value: "WebhookX/" + config.VERSION},
	}
	DefaultDelivererRequestHeaders = []Header{
		{Name: "User-Agent", Value: "WebhookX/" + config.VERSION},
		{Name: "Content-Type", Value: "application/json; charset=utf-8"},
	}
)

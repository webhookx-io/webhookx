package constants

import (
	"github.com/webhookx-io/webhookx/config"
	"strings"
	"time"
)

// Task Queue
const (
	TaskQueueName                  = "webhookx:queue"
	TaskQueueInvisibleQueueName    = "webhookx:queue_invisible"
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

const (
	RequeueBatch    = 20
	RequeueInterval = time.Second * 60
)

type CacheKey string

func (c CacheKey) Build(id string) string {
	var sb strings.Builder
	sb.WriteString(Namespace)
	sb.WriteString(":")
	sb.WriteString(string(c))
	sb.WriteString(":")
	sb.WriteString(id)
	return sb.String()
}

const (
	Namespace             string   = "webhookx"
	EventCacheKey         CacheKey = "events"
	EndpointCacheKey      CacheKey = "endpoints"
	EndpointPluginsKey    CacheKey = "endpoint_plugins"
	SourcePluginsKey      CacheKey = "source_plugins"
	SourceCacheKey        CacheKey = "sources"
	WorkspaceCacheKey     CacheKey = "workspaces"
	AttemptCacheKey       CacheKey = "attempts"
	PluginCacheKey        CacheKey = "plugins"
	AttemptDetailCacheKey CacheKey = "attempt_details"
	WorkspaceEndpointsKey CacheKey = "workspaces_endpoints"
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

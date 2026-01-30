package services

import (
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/ratelimiter"
	"github.com/webhookx-io/webhookx/services/eventbus"
	"github.com/webhookx-io/webhookx/services/schedule"
	"github.com/webhookx-io/webhookx/services/task"
)

type Services struct {
	Scheduler   schedule.Scheduler
	EventBus    eventbus.EventBus
	Metrics     *metrics.Metrics
	Task        *task.TaskService
	RateLimiter ratelimiter.RateLimiter
}

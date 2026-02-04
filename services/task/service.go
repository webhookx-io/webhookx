package task

import (
	"context"
	"time"

	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"go.uber.org/zap"
)

type TaskService struct {
	log    *zap.SugaredLogger
	db     *db.DB
	queue  taskqueue.TaskQueue
	notify chan struct{}
}

func NewTaskService(log *zap.SugaredLogger, db *db.DB, queue taskqueue.TaskQueue) *TaskService {
	return &TaskService{
		log:    log,
		db:     db,
		queue:  queue,
		notify: make(chan struct{}, 1),
	}
}

func (s *TaskService) ScheduleAttempts(ctx context.Context, attempts []*entities.Attempt) {
	if len(attempts) == 0 {
		return
	}

	now := time.Now()
	notify := false
	maxScheduleAt := now.Add(constants.TaskQueuePreScheduleTimeWindow)
	tasks := make([]*taskqueue.TaskMessage, 0)
	ids := make([]string, 0)
	for _, attempt := range attempts {
		if attempt.ScheduledAt.Before(maxScheduleAt) {
			if !notify && attempt.ScheduledAt.Before(now) {
				notify = true
			}
			tasks = append(tasks, &taskqueue.TaskMessage{
				ID:          attempt.ID,
				ScheduledAt: attempt.ScheduledAt.Time,
				Data: &taskqueue.MessageData{
					EventID:    attempt.EventId,
					EndpointId: attempt.EndpointId,
					Attempt:    attempt.AttemptNumber,
					Event:      string(attempt.Event.Data),
				},
			})
			ids = append(ids, attempt.ID)
		}
	}

	if len(tasks) == 0 {
		return
	}

	ctx, span := tracing.Start(ctx, "attempt.schedule")
	defer span.End()

	err := s.queue.Add(ctx, tasks)
	if err != nil {
		s.log.Warnf("failed to add tasks to queue: %v", err)
		return
	}
	err = s.db.Attempts.UpdateStatusToQueued(ctx, ids)
	if err != nil {
		s.log.Warnf("failed to update attempts status: %v", err)
	}

	if notify {
		s.Notify()
	}
}

func (s *TaskService) Notify() {
	select {
	case s.notify <- struct{}{}:
	default:
	}
}

func (s *TaskService) NotificationChannel() <-chan struct{} {
	return s.notify
}

func (s *TaskService) GetTasks(ctx context.Context, opts *taskqueue.GetOptions) ([]*taskqueue.TaskMessage, error) {
	return s.queue.Get(ctx, opts)
}

func (s *TaskService) DeleteTask(ctx context.Context, task *taskqueue.TaskMessage) error {
	return s.queue.Delete(ctx, task.ID)
}

func (s *TaskService) DeleteTasks(ctx context.Context, ids []string) error {
	return s.queue.Delete(ctx, ids...)
}

func (s *TaskService) ScheduleTask(ctx context.Context, id string, scheduledAt time.Time) error {
	return s.queue.Schedule(ctx, id, scheduledAt)
}

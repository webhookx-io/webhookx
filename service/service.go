package service

import (
	"context"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"go.uber.org/zap"
)

type Service struct {
	log   *zap.SugaredLogger
	db    *db.DB
	queue taskqueue.TaskQueue
}

type Options struct {
	DB        *db.DB
	TaskQueue taskqueue.TaskQueue
}

func NewService(opts Options) *Service {
	return &Service{
		log:   zap.S(),
		db:    opts.DB,
		queue: opts.TaskQueue,
	}
}

func (s *Service) ScheduleAttempts(ctx context.Context, attempts []*entities.Attempt) {
	if len(attempts) == 0 {
		return
	}

	tasks := make([]*taskqueue.TaskMessage, 0)
	ids := make([]string, 0)
	for _, attempt := range attempts {
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

	if len(tasks) == 0 {
		return
	}

	err := s.queue.Add(ctx, tasks)
	if err != nil {
		s.log.Warnf("failed to add tasks to queue: %v", err)
		return
	}
	err = s.db.Attempts.UpdateStatusToQueued(ctx, ids)
	if err != nil {
		s.log.Warnf("failed to update attempts status: %v", err)
	}
}

func (s *Service) GetTasks(ctx context.Context, opts *taskqueue.GetOptions) ([]*taskqueue.TaskMessage, error) {
	return s.queue.Get(ctx, opts)
}

func (s *Service) DeleteTask(ctx context.Context, task *taskqueue.TaskMessage) error {
	return s.queue.Delete(ctx, task)
}

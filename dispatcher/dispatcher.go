package dispatcher

import (
	"context"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"time"
)

// Dispatcher is Event Dispatcher
type Dispatcher struct {
	log   *zap.SugaredLogger
	queue queue.TaskQueue
	db    *db.DB
}

func NewDispatcher(log *zap.SugaredLogger, queue queue.TaskQueue, db *db.DB) *Dispatcher {
	dispatcher := &Dispatcher{
		log:   log,
		queue: queue,
		db:    db,
	}
	return dispatcher
}

func (d *Dispatcher) Dispatch(ctx context.Context, event *entities.Event) error {
	endpoints, err := listSubscribedEndpoints(ctx, d.db, event.EventType)
	if err != nil {
		return err
	}

	return d.dispatch(ctx, event, endpoints)
}

func (d *Dispatcher) DispatchEndpoint(ctx context.Context, eventId string, endpoints []*entities.Endpoint) error {
	attempts := make([]*entities.Attempt, 0, len(endpoints))
	tasks := make([]*queue.TaskMessage, 0, len(endpoints))

	now := time.Now()
	for _, endpoint := range endpoints {
		delay := endpoint.Retry.Config.Attempts[0]
		attempt := &entities.Attempt{
			ID:            utils.KSUID(),
			EventId:       eventId,
			EndpointId:    endpoint.ID,
			Status:        entities.AttemptStatusInit,
			AttemptNumber: 1,
			ScheduledAt:   types.NewTime(now.Add(time.Second * time.Duration(delay))),
			TriggerMode:   entities.AttemptTriggerModeManual,
		}

		task := &queue.TaskMessage{
			ID: attempt.ID,
			Data: &model.MessageData{
				EventID:    eventId,
				EndpointId: endpoint.ID,
				Attempt:    1,
			},
		}
		attempts = append(attempts, attempt)
		tasks = append(tasks, task)
	}

	err := d.db.AttemptsWS.BatchInsert(ctx, attempts)
	if err != nil {
		return err
	}

	for i, task := range tasks {
		err := d.queue.Add(task, attempts[i].ScheduledAt.Time)
		if err != nil {
			d.log.Warnf("failed to add task to queue: %v", err)
			continue
		}
		err = d.db.AttemptsWS.UpdateStatus(ctx, task.ID, entities.AttemptStatusQueued)
		if err != nil {
			d.log.Warnf("failed to update attempt status: %v", err)
		}
	}

	return nil
}

func (d *Dispatcher) dispatch(ctx context.Context, event *entities.Event, endpoints []*entities.Endpoint) error {
	attempts := make([]*entities.Attempt, 0, len(endpoints))
	tasks := make([]*queue.TaskMessage, 0, len(endpoints))

	err := d.db.TX(ctx, func(ctx context.Context) error {
		now := time.Now()
		err := d.db.Events.Insert(ctx, event)
		if err != nil {
			return err
		}

		for _, endpoint := range endpoints {
			delay := endpoint.Retry.Config.Attempts[0]
			attempt := &entities.Attempt{
				ID:            utils.KSUID(),
				EventId:       event.ID,
				EndpointId:    endpoint.ID,
				Status:        entities.AttemptStatusInit,
				AttemptNumber: 1,
				ScheduledAt:   types.NewTime(now.Add(time.Second * time.Duration(delay))),
				TriggerMode:   entities.AttemptTriggerModeInitial,
			}

			task := &queue.TaskMessage{
				ID: attempt.ID,
				Data: &model.MessageData{
					EventID:    event.ID,
					EndpointId: endpoint.ID,
					Attempt:    1,
				},
			}
			attempts = append(attempts, attempt)
			tasks = append(tasks, task)
		}

		return d.db.AttemptsWS.BatchInsert(ctx, attempts)
	})
	if err != nil {
		return err
	}

	for i, task := range tasks {
		err := d.queue.Add(task, attempts[i].ScheduledAt.Time)
		if err != nil {
			d.log.Warnf("failed to add task to queue: %v", err)
			continue
		}
		err = d.db.AttemptsWS.UpdateStatus(ctx, task.ID, entities.AttemptStatusQueued)
		if err != nil {
			d.log.Warnf("failed to update attempt status: %v", err)
		}
	}

	return nil
}

func listSubscribedEndpoints(ctx context.Context, db *db.DB, eventType string) (list []*entities.Endpoint, err error) {
	var q query.EndpointQuery
	endpoints, err := db.EndpointsWS.List(ctx, &q)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints {
		if !endpoint.Enabled {
			continue
		}
		for _, event := range endpoint.Events {
			if eventType == event {
				list = append(list, endpoint)
			}
		}
	}

	return
}

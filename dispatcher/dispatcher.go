package dispatcher

import (
	"context"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"time"
)

// Dispatcher is Event Dispatcher
type Dispatcher struct {
	log   *zap.SugaredLogger
	queue taskqueue.TaskQueue
	db    *db.DB
}

func NewDispatcher(log *zap.SugaredLogger, queue taskqueue.TaskQueue, db *db.DB) *Dispatcher {
	dispatcher := &Dispatcher{
		log:   log,
		queue: queue,
		db:    db,
	}
	return dispatcher
}

func (d *Dispatcher) Dispatch(ctx context.Context, event *entities.Event) error {
	endpoints, err := d.listSubscribedEndpoint(ctx, event.WorkspaceId, event.EventType)
	if err != nil {
		return err
	}

	attempts := fanout(event, endpoints, entities.AttemptTriggerModeInitial)
	if len(attempts) == 0 {
		return d.db.Events.Insert(ctx, event)
	}

	err = d.db.TX(ctx, func(ctx context.Context) error {
		err := d.db.Events.Insert(ctx, event)
		if err != nil {
			return err
		}
		return d.db.Attempts.BatchInsert(ctx, attempts)
	})
	if err != nil {
		return err
	}

	go d.sendToQueue(context.TODO(), attempts)

	return nil
}

func (d *Dispatcher) DispatchBatch(ctx context.Context, events []*entities.Event) error {
	if len(events) == 0 {
		return nil
	}

	eventAttempts := make(map[string][]*entities.Attempt)
	for _, event := range events {
		endpoints, err := d.listSubscribedEndpoint(ctx, event.WorkspaceId, event.EventType)
		if err != nil {
			return err
		}
		eventAttempts[event.ID] = fanout(event, endpoints, entities.AttemptTriggerModeInitial)
	}

	attempts := make([]*entities.Attempt, 0)
	err := d.db.TX(ctx, func(ctx context.Context) error {
		ids, err := d.db.Events.BatchInsertIgnoreConflict(ctx, events)
		if err != nil {
			return err
		}
		for _, id := range ids {
			attempts = append(attempts, eventAttempts[id]...)
		}
		return d.db.Attempts.BatchInsert(ctx, attempts)
	})

	go d.sendToQueue(context.TODO(), attempts)

	return err
}

func fanout(event *entities.Event, endpoints []*entities.Endpoint, mode entities.AttemptTriggerMode) []*entities.Attempt {
	attempts := make([]*entities.Attempt, 0, len(endpoints))
	now := time.Now()
	for _, endpoint := range endpoints {
		delay := endpoint.Retry.Config.Attempts[0]
		attempt := &entities.Attempt{
			ID:            utils.KSUID(),
			EventId:       event.ID,
			EndpointId:    endpoint.ID,
			Status:        entities.AttemptStatusInit,
			AttemptNumber: 1,
			ScheduledAt:   types.NewTime(now.Add(time.Second * time.Duration(delay))),
			TriggerMode:   mode,
		}
		attempt.WorkspaceId = event.WorkspaceId
		attempts = append(attempts, attempt)
	}
	return attempts
}

func (d *Dispatcher) DispatchEndpoint(ctx context.Context, event *entities.Event, endpoints []*entities.Endpoint) error {
	attempts := fanout(event, endpoints, entities.AttemptTriggerModeManual)

	err := d.db.Attempts.BatchInsert(ctx, attempts)
	if err != nil {
		return err
	}

	d.sendToQueue(context.TODO(), attempts)

	return nil
}

func (d *Dispatcher) sendToQueue(ctx context.Context, attempts []*entities.Attempt) {
	for _, attempt := range attempts {
		task := &taskqueue.TaskMessage{
			ID: attempt.ID,
			Data: &model.MessageData{
				EventID:    attempt.EventId,
				EndpointId: attempt.EndpointId,
				Attempt:    attempt.AttemptNumber,
			},
		}
		err := d.queue.Add(ctx, task, attempt.ScheduledAt.Time)
		if err != nil {
			d.log.Warnf("failed to add task to queue: %v", err)
			continue
		}
		err = d.db.Attempts.UpdateStatus(ctx, task.ID, entities.AttemptStatusQueued)
		if err != nil {
			d.log.Warnf("failed to update attempt status: %v", err)
		}
	}
}

func (d *Dispatcher) listSubscribedEndpoint(ctx context.Context, wid, eventType string) (list []*entities.Endpoint, err error) {
	var q query.EndpointQuery
	q.WorkspaceId = &wid
	endpoints, err := d.db.Endpoints.List(ctx, &q)
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

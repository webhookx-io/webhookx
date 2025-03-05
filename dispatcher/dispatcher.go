package dispatcher

import (
	"context"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"time"
)

// Dispatcher is Event Dispatcher
type Dispatcher struct {
	log     *zap.SugaredLogger
	queue   taskqueue.TaskQueue
	db      *db.DB
	metrics *metrics.Metrics
	bus     *eventbus.EventBus
}

func NewDispatcher(log *zap.SugaredLogger, queue taskqueue.TaskQueue, db *db.DB, metrics *metrics.Metrics, bus *eventbus.EventBus) *Dispatcher {
	dispatcher := &Dispatcher{
		log:     log,
		queue:   queue,
		db:      db,
		metrics: metrics,
		bus:     bus,
	}
	dispatcher.bus.Subscribe("endpoint.crud", func(v interface{}) {
		data := v.(*eventbus.CrudData)
		err := mcache.Invalidate(context.TODO(), constants.WorkspaceEndpointsKey.Build(data.WID))
		if err != nil {
			log.Errorf("failed to invalidate cache: key=%s %v", constants.WorkspaceEndpointsKey.Build(data.WID), err)
		}
	})
	return dispatcher
}

func (d *Dispatcher) Dispatch(ctx context.Context, event *entities.Event) error {
	return d.DispatchBatch(ctx, []*entities.Event{event})
}

func (d *Dispatcher) DispatchBatch(ctx context.Context, events []*entities.Event) error {
	n, err := d.dispatchBatch(ctx, events)
	if d.metrics.Enabled && err == nil {
		d.metrics.EventPersistCounter.Add(float64(n))
	}
	return err
}

func (d *Dispatcher) dispatchBatch(ctx context.Context, events []*entities.Event) (int, error) {
	ctx, span := tracing.Start(ctx, "dispatcher.dispatch", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(events) == 0 {
		return 0, nil
	}

	maps := make(map[string][]*entities.Attempt)
	for _, event := range events {
		endpoints, err := d.listSubscribedEndpoint(ctx, event.WorkspaceId, event.EventType)
		if err != nil {
			return 0, err
		}
		if len(endpoints) != 0 {
			maps[event.ID] = fanout(event, endpoints, entities.AttemptTriggerModeInitial)
		}
	}

	if len(maps) == 0 {
		ids, err := d.db.Events.BatchInsertIgnoreConflict(ctx, events)
		return len(ids), err
	}

	attempts := make([]*entities.Attempt, 0)
	n := 0
	err := d.db.TX(ctx, func(ctx context.Context) error {
		ids, err := d.db.Events.BatchInsertIgnoreConflict(ctx, events)
		if err != nil {
			return err
		}
		n = len(ids)
		for _, id := range ids {
			attempts = append(attempts, maps[id]...)
		}
		return d.db.Attempts.BatchInsert(ctx, attempts)
	})
	if err == nil {
		go d.sendToQueue(context.WithoutCancel(ctx), attempts)
	}
	return n, err
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

	d.sendToQueue(ctx, attempts)

	return nil
}

func (d *Dispatcher) sendToQueue(ctx context.Context, attempts []*entities.Attempt) {
	tasks := make([]*taskqueue.TaskMessage, 0, len(attempts))
	ids := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		tasks = append(tasks, &taskqueue.TaskMessage{
			ID:          attempt.ID,
			ScheduledAt: attempt.ScheduledAt.Time,
			Data: &model.MessageData{
				EventID:    attempt.EventId,
				EndpointId: attempt.EndpointId,
				Attempt:    attempt.AttemptNumber,
			},
		})
		ids = append(ids, attempt.ID)
	}

	err := d.queue.Add(ctx, tasks)
	if err != nil {
		d.log.Warnf("failed to add tasks to queue: %v", err)
		return
	}
	err = d.db.Attempts.UpdateStatusBatch(ctx, entities.AttemptStatusQueued, ids)
	if err != nil {
		d.log.Warnf("failed to update attempts status: %v", err)
	}
}

func (d *Dispatcher) listSubscribedEndpoint(ctx context.Context, wid, eventType string) (list []*entities.Endpoint, err error) {
	endpoints, err := listWorkspaceEndpoints(ctx, d.db, wid)
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

func listWorkspaceEndpoints(ctx context.Context, db *db.DB, wid string) ([]*entities.Endpoint, error) {
	// refactor me
	cacheKey := constants.WorkspaceEndpointsKey.Build(wid)
	endpoints, err := mcache.Load(ctx, cacheKey, nil, func(ctx context.Context, id string) (*[]*entities.Endpoint, error) {
		var q query.EndpointQuery
		q.WorkspaceId = &wid
		//q.Enabled = true
		endpoints, err := db.Endpoints.List(ctx, &q)
		if err != nil {
			return nil, err
		}
		return &endpoints, nil
	}, wid)
	return *endpoints, err
}

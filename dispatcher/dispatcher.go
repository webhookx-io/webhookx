package dispatcher

import (
	"context"
	"slices"
	"time"

	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Dispatcher is Event Dispatcher
type Dispatcher struct {
	opts     Options
	db       *db.DB
	log      *zap.SugaredLogger
	registry *Registry
}

type Options struct {
	DB       *db.DB
	Metrics  *metrics.Metrics
	Registry *Registry
}

func NewDispatcher(opts Options) *Dispatcher {
	dispatcher := &Dispatcher{
		db:       opts.DB,
		log:      zap.S(),
		opts:     opts,
		registry: opts.Registry,
	}
	return dispatcher
}

func (d *Dispatcher) Dispatch(ctx context.Context, events []*entities.Event) ([]*entities.Attempt, error) {
	ctx, span := tracing.Start(ctx, "dispatcher.dispatch", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	if len(events) == 0 {
		return nil, nil
	}

	uids := make([]string, 0)
	maps := make(map[string][]*entities.Attempt)
	for _, event := range events {
		endpoints, err := d.registry.LookUp(ctx, event)
		if err != nil {
			return nil, err
		}
		if len(endpoints) != 0 {
			attempts := fanout(event, endpoints, entities.AttemptTriggerModeInitial)
			maps[event.ID] = attempts
		}
		if event.UniqueId != nil {
			uids = append(uids, *event.UniqueId)
		}
	}

	if len(uids) > 0 {
		exists, err := d.db.Events.ListUniqueIds(ctx, uids)
		if err != nil {
			return nil, err
		}
		if len(exists) > 0 {
			for _, id := range exists {
				delete(maps, id)
			}
			filtered := events[:0]
			for _, event := range events {
				if event.UniqueId == nil || !slices.Contains(exists, *event.UniqueId) {
					filtered = append(filtered, event)
				}
			}
			events = filtered
		}
	}

	if len(maps) == 0 {
		ids, err := d.db.Events.BatchInsertIgnoreConflict(ctx, events)
		if d.opts.Metrics.Enabled {
			d.opts.Metrics.EventPersistCounter.Add(float64(len(ids)))
		}
		return nil, err
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
		if d.opts.Metrics.Enabled {
			d.opts.Metrics.EventPersistCounter.Add(float64(n))
		}
	}
	return attempts, err
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
			Event:         event,
		}
		attempt.WorkspaceId = event.WorkspaceId
		attempts = append(attempts, attempt)
	}
	return attempts
}

func (d *Dispatcher) DispatchEndpoint(ctx context.Context, event *entities.Event, endpoints []*entities.Endpoint) ([]*entities.Attempt, error) {
	attempts := fanout(event, endpoints, entities.AttemptTriggerModeManual)

	err := d.db.Attempts.BatchInsert(ctx, attempts)
	if err != nil {
		return nil, err
	}

	return attempts, nil
}

func (d *Dispatcher) WarmUp() {
	t := time.Now()
	err := d.registry.Warmup()
	if err != nil {
		d.log.Warnf("failed to warm-up: %v", err)
		return
	}
	d.log.Debugf("warm-up finished in %dms", time.Since(t).Milliseconds())
}

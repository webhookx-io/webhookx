package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/pool"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/service"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"maps"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	counter    atomic.Int64
	failures   atomic.Int64
	processing atomic.Int64
)

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc

	opts Options

	log *zap.SugaredLogger

	deliverer deliverer.Deliverer
	db        *db.DB
	tracer    *tracing.Tracer
	pool      *pool.Pool
	metrics   *metrics.Metrics
	srv       *service.Service
}

type Options struct {
	RequeueJobBatch    int
	RequeueJobInterval time.Duration
	PoolSize           int
	PoolConcurrency    int

	DB          *db.DB
	Deliverer   deliverer.Deliverer
	Metrics     *metrics.Metrics
	Tracer      *tracing.Tracer
	EventBus    eventbus.Bus
	Srv         *service.Service
	RedisClient *redis.Client
}

func init() {
	stats.Register(stats.ProviderFunc(func() map[string]interface{} {
		return map[string]interface{}{
			"outbound.requests":            counter.Load(),
			"outbound.failed_requests":     failures.Load(),
			"outbound.processing_requests": processing.Load(),
		}
	}))
}

func NewWorker(opts Options) *Worker {
	opts.RequeueJobBatch = utils.DefaultIfZero(opts.RequeueJobBatch, constants.RequeueBatch)
	opts.RequeueJobInterval = utils.DefaultIfZero(opts.RequeueJobInterval, constants.RequeueInterval)
	opts.PoolSize = utils.DefaultIfZero(opts.PoolSize, 10000)
	opts.PoolConcurrency = utils.DefaultIfZero(opts.PoolConcurrency, runtime.NumCPU()*100)

	ctx, cancel := context.WithCancel(context.Background())
	worker := &Worker{
		ctx:       ctx,
		cancel:    cancel,
		opts:      opts,
		log:       zap.S().Named("worker"),
		deliverer: opts.Deliverer,
		db:        opts.DB,
		pool:      pool.NewPool(opts.PoolSize, opts.PoolConcurrency),
		metrics:   opts.Metrics,
		tracer:    opts.Tracer,
		srv:       opts.Srv,
	}

	worker.registerEventHandler(opts.EventBus)

	return worker
}

func (w *Worker) registerEventHandler(bus eventbus.Bus) {
	rs := redsync.New(goredis.NewPool(w.opts.RedisClient))
	bus.Subscribe("plugin.crud", func(data interface{}) {
		plugin := entities.Plugin{}
		if err := json.Unmarshal(data.(*eventbus.CrudData).Data, &plugin); err != nil {
			w.log.Errorf("failed to unmarshal event data: %s", err)
			return
		}

		if plugin.EndpointId != nil {
			cacheKey := constants.EndpointPluginsKey.Build(*plugin.EndpointId)
			err := mcache.Invalidate(context.TODO(), cacheKey)
			if err != nil {
				w.log.Errorf("failed to invalidate cache: key=%s %v", cacheKey, err)
			}
		}
	})
	bus.ClusteringSubscribe(eventbus.EventEventFanout, func(data []byte) {
		eventData := &eventbus.EventFanoutData{}
		if err := json.Unmarshal(data, eventData); err != nil {
			w.log.Errorf("failed to unmarshal event: %s", err)
			return
		}
		bus.Broadcast(eventbus.EventEventFanout, eventData)
	})
	bus.Subscribe(eventbus.EventEventFanout, func(data interface{}) {
		ctx := context.TODO()
		fanoutData := data.(*eventbus.EventFanoutData)
		if len(fanoutData.AttemptIds) == 0 {
			return
		}

		mux := rs.NewMutex("lock:event.fanout:" + fanoutData.EventId)
		if err := mux.TryLock(); err != nil {
			w.log.Errorf("failed to acquire distributed lock '%s' %s", mux.Name(), err)
			return
		}
		defer func() { _, _ = mux.Unlock() }()

		q := query.AttemptQuery{}
		q.IDs = fanoutData.AttemptIds
		q.Status = utils.Pointer(entities.AttemptStatusInit)
		attempts, err := w.db.Attempts.List(ctx, &q)
		if err != nil {
			w.log.Errorf("failed to list attempts: id=%s err=%s", fanoutData.EventId, err)
			return
		}

		if len(attempts) == 0 {
			return
		}

		event, err := w.db.Events.Get(ctx, fanoutData.EventId)
		if err != nil {
			w.log.Errorf("failed to get event: id=%s err=%s", fanoutData.EventId, err)
			return
		}

		for _, e := range attempts {
			e.Event = event
		}

		w.srv.ScheduleAttempts(ctx, attempts)
	})
}

func (w *Worker) run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	options := &taskqueue.GetOptions{
		Count: 20,
	}
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			for {
				tasks, err := w.srv.GetTasks(context.TODO(), options)
				if err != nil {
					w.log.Errorf("failed to fetch tasks from queue: %v", err)
					break
				}
				if len(tasks) == 0 {
					break
				}

				w.log.Debugf("received tasks: %d", len(tasks))
				var errs []error
				for _, task := range tasks {
					err = w.pool.SubmitFn(time.Second*5, func() {
						processing.Add(1)
						defer processing.Add(-1)

						// TODO: start trace with task Context
						ctx := context.TODO()
						if w.tracer != nil {
							tracingCtx, span := w.tracer.Start(ctx, "worker.submit", trace.WithSpanKind(trace.SpanKindServer))
							defer span.End()
							ctx = tracingCtx
						}
						task.Data = &taskqueue.MessageData{}
						err = task.UnmarshalData(task.Data)
						if err != nil {
							w.log.Errorf("failed to unmarshal task: %v", err)
							_ = w.srv.DeleteTask(ctx, task)
							return
						}

						err = w.handleTask(ctx, task)
						if err != nil {
							// TODO: delete task when causes error too many times (maxReceiveCount)
							w.log.Errorf("failed to handle task: %v", err)
							return
						}

						_ = w.srv.DeleteTask(ctx, task)
					})
					if err != nil {
						if errors.Is(err, pool.ErrPoolTernimated) {
							return // worker is shutting down
						}
						errs = append(errs, err)
					}
				}
				if len(errs) > 0 { // pool.ErrTimeout
					w.log.Warnf("failed to submit tasks to pool: %v", errs) // consider tuning pool configuration
					break
				}
			}
		}
	}
}

// Start starts worker
func (w *Worker) Start() error {
	w.log.Infow("starting worker", zap.Any("pool", map[string]interface{}{
		"size":      w.opts.PoolSize,
		"consumers": w.opts.PoolConcurrency,
	}))

	go w.run()

	schedule.Schedule(w.ctx, w.ProcessRequeue, w.opts.RequeueJobInterval)
	return nil
}

// Stop stops worker
func (w *Worker) Stop() error {
	w.cancel()
	w.log.Named("pool").Infow("closing pool", "handling", w.pool.GetHandling())
	w.pool.Shutdown()
	w.log.Named("pool").Info("closed pool")
	w.log.Info("worker stopped")

	return nil
}

func (w *Worker) ProcessRequeue() {
	batchSize := w.opts.RequeueJobBatch

	var done bool
	for {
		err := w.db.TX(context.TODO(), func(ctx context.Context) error {
			maxScheduledAt := time.Now().Add(constants.TaskQueuePreScheduleTimeWindow)
			attempts, err := w.db.Attempts.ListUnqueuedForUpdate(ctx, maxScheduledAt, batchSize)
			if err != nil {
				return err
			}

			if len(attempts) > 0 {
				for _, attempt := range attempts {
					event, err := w.db.Events.Get(ctx, attempt.EventId)
					if err != nil {
						return err
					}
					attempt.Event = event
				}
				w.srv.ScheduleAttempts(ctx, attempts)
			}

			if len(attempts) < batchSize {
				done = true
			}

			return nil
		})

		if err != nil {
			w.log.Error(err)
			return
		}

		if done {
			break
		}
	}
}

func (w *Worker) handleTask(ctx context.Context, task *taskqueue.TaskMessage) error {
	if w.tracer != nil {
		tracingCtx, span := w.tracer.Start(ctx, "worker.handle_task", trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		ctx = tracingCtx
	}
	data := task.Data.(*taskqueue.MessageData)

	// verify endpoint
	cacheKey := constants.EndpointCacheKey.Build(data.EndpointId)
	endpoint, err := mcache.Load(ctx, cacheKey, nil, w.db.Endpoints.Get, data.EndpointId)
	if err != nil {
		return err
	}
	if endpoint == nil {
		return w.db.Attempts.UpdateErrorCode(ctx, task.ID, entities.AttemptStatusCanceled, entities.AttemptErrorCodeEndpointNotFound)
	}
	if !endpoint.Enabled {
		return w.db.Attempts.UpdateErrorCode(ctx, task.ID, entities.AttemptStatusCanceled, entities.AttemptErrorCodeEndpointDisabled)
	}

	if data.Event == "" { // backward compatibility
		// verify event
		cacheKey = constants.EventCacheKey.Build(data.EventID)
		opts := &mcache.LoadOptions{DisableLRU: true}
		event, err := mcache.Load(ctx, cacheKey, opts, w.db.Events.Get, data.EventID)
		if err != nil {
			return err
		}
		if event == nil {
			return w.db.Attempts.UpdateErrorCode(ctx, task.ID, entities.AttemptStatusCanceled, entities.AttemptErrorCodeUnknown)
		}
		data.Event = string(event.Data)
	}

	plugins, err := listEndpointPlugins(ctx, w.db, endpoint.ID)
	if err != nil {
		return err
	}

	cacheKey = constants.WorkspaceCacheKey.Build(endpoint.WorkspaceId)
	//workspace, err := mcache.Load(ctx, cacheKey, nil, w.DB.Workspaces.Get, endpoint.WorkspaceId)
	//if err != nil {
	//	return err
	//}

	outbound := plugin.Outbound{
		URL:     endpoint.Request.URL,
		Method:  endpoint.Request.Method,
		Headers: make(map[string]string),
		Payload: data.Event,
	}
	maps.Copy(outbound.Headers, endpoint.Request.Headers)
	pluginCtx := &plugin.Context{
		//Workspace: workspace,
	}
	for _, p := range plugins {
		executor, err := p.Plugin()
		if err != nil {
			return err
		}

		err = executor.ExecuteOutbound(&outbound, pluginCtx)
		if err != nil {
			return fmt.Errorf("failed to execute %s plugin: %v", p.Name, err)
		}
	}

	request := &deliverer.Request{
		Request: nil,
		URL:     outbound.URL,
		Method:  outbound.Method,
		Payload: []byte(outbound.Payload),
		Headers: outbound.Headers,
		Timeout: time.Duration(endpoint.Request.Timeout) * time.Millisecond,
	}

	// deliver the request
	startAt := time.Now()
	ctx, span := tracing.Start(ctx, "worker.deliver", trace.WithSpanKind(trace.SpanKindClient))
	response := w.deliverer.Deliver(ctx, request)
	span.End()
	finishAt := time.Now()

	if response.Error != nil {
		w.log.Infof("failed to delivery event: %v", response.Error)
	}
	w.log.Debugf("delivery response: %v", response)

	result := buildAttemptResult(request, response)
	result.AttemptedAt = types.NewTime(startAt)
	result.Exhausted = data.Attempt >= len(endpoint.Retry.Config.Attempts)

	counter.Add(1)
	if result.Status == entities.AttemptStatusFailure {
		failures.Add(1)
	}

	if w.metrics.Enabled {
		w.metrics.AttemptTotalCounter.Add(1)
		if result.Status == entities.AttemptStatusFailure {
			w.metrics.AttemptFailedCounter.Add(1)
		}
		w.metrics.AttemptResponseDurationHistogram.Observe(response.Latancy.Seconds())
	}

	err = w.db.Attempts.UpdateDelivery(ctx, task.ID, result)
	if err != nil {
		return err
	}

	go func() {
		attemptDetail := &entities.AttemptDetail{
			ID:             task.ID,
			RequestHeaders: utils.HeaderMap(request.Request.Header),
			RequestBody:    utils.Pointer(string(request.Payload)),
		}
		if len(response.Header) > 0 {
			attemptDetail.ResponseHeaders = utils.Pointer(entities.Headers(utils.HeaderMap(response.Header)))
		}
		if response.ResponseBody != nil {
			attemptDetail.ResponseBody = utils.Pointer(string(response.ResponseBody))
		}
		attemptDetail.WorkspaceId = endpoint.WorkspaceId
		err = w.db.AttemptDetails.Insert(ctx, attemptDetail)
		if err != nil {
			w.log.Errorf("failed to insert attempt detail: %v", err)
		}
	}()

	if result.Status == entities.AttemptStatusSuccess {
		return nil
	}

	if result.Exhausted {
		w.log.Debugf("webhook delivery exhausted : %s", task.ID)
		return nil
	}

	delay := endpoint.Retry.Config.Attempts[data.Attempt]
	nextAttempt := &entities.Attempt{
		ID:            utils.KSUID(),
		EventId:       data.EventID,
		EndpointId:    endpoint.ID,
		Status:        entities.AttemptStatusInit,
		AttemptNumber: data.Attempt + 1,
		ScheduledAt:   types.NewTime(finishAt.Add(time.Second * time.Duration(delay))),
		TriggerMode:   entities.AttemptTriggerModeAutomatic,
		Event:         &entities.Event{ID: data.EventID, Data: json.RawMessage(data.Event)},
	}
	nextAttempt.WorkspaceId = endpoint.WorkspaceId

	err = w.db.Attempts.Insert(ctx, nextAttempt)
	if err != nil {
		return err
	}

	w.srv.ScheduleAttempts(ctx, []*entities.Attempt{nextAttempt})
	return nil
}

func buildAttemptResult(request *deliverer.Request, response *deliverer.Response) *dao.AttemptResult {
	result := &dao.AttemptResult{
		Request: &entities.AttemptRequest{
			URL:    request.URL,
			Method: request.Method,
		},
		Status: entities.AttemptStatusSuccess,
	}

	if response.Error != nil {
		if errors.Is(response.Error, context.DeadlineExceeded) {
			result.ErrorCode = utils.Pointer(entities.AttemptErrorCodeTimeout)
		} else {
			result.ErrorCode = utils.Pointer(entities.AttemptErrorCodeUnknown)
		}
	}

	if !response.Is2xx() {
		result.Status = entities.AttemptStatusFailure
	}

	if response.StatusCode != 0 {
		result.Response = &entities.AttemptResponse{
			Status:  response.StatusCode,
			Latency: response.Latancy.Milliseconds(),
		}
	}

	return result
}

func listEndpointPlugins(ctx context.Context, db *db.DB, endpointId string) ([]*entities.Plugin, error) {
	// refactor me
	cacheKey := constants.EndpointPluginsKey.Build(endpointId)
	plugins, err := mcache.Load(ctx, cacheKey, nil, func(ctx context.Context, id string) (*[]*entities.Plugin, error) {
		plugins, err := db.Plugins.ListEndpointPlugin(ctx, id)
		if err != nil {
			return nil, err
		}
		return &plugins, nil
	}, endpointId)
	if err != nil {
		return nil, err
	}
	return *plugins, err
}

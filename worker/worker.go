package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/batchqueue"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/pool"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins"
	"github.com/webhookx-io/webhookx/services"
	"github.com/webhookx-io/webhookx/services/eventbus"
	"github.com/webhookx-io/webhookx/services/schedule"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	counter    atomic.Int64
	failures   atomic.Int64
	processing atomic.Int64
)

var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrTerminated        = errors.New("terminated")
)

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc

	opts Options

	log *zap.SugaredLogger

	services *services.Services

	deliverer       deliverer.Deliverer
	db              *db.DB
	pool            *pool.Pool[*taskqueue.TaskMessage]
	queueRequestLog *batchqueue.BatchQueue[*entities.AttemptDetail]
}

type Options struct {
	RequeueJobBatch int
	PoolSize        int
	PoolConcurrency int

	DB               *db.DB
	DelivererOptions deliverer.Options
	RedisClient      *redis.Client
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

func NewWorker(opts Options, services *services.Services) *Worker {
	opts.RequeueJobBatch = utils.DefaultIfZero(opts.RequeueJobBatch, 50)
	opts.PoolSize = utils.DefaultIfZero(opts.PoolSize, 10000)
	opts.PoolConcurrency = utils.DefaultIfZero(opts.PoolConcurrency, runtime.NumCPU()*100)

	ctx, cancel := context.WithCancel(context.Background())
	worker := &Worker{
		ctx:             ctx,
		cancel:          cancel,
		opts:            opts,
		log:             zap.S().Named("worker"),
		db:              opts.DB,
		services:        services,
		queueRequestLog: batchqueue.New[*entities.AttemptDetail]("request_log", 1000, 50, time.Millisecond*500),
	}

	worker.pool = pool.New[*taskqueue.TaskMessage](
		opts.PoolSize,
		opts.PoolConcurrency,
		pool.HandlerFunc[*taskqueue.TaskMessage](worker.runTask),
	)

	worker.registerEventHandler(services.EventBus)

	return worker
}

func (w *Worker) Name() string {
	return "worker"
}

func (w *Worker) submitTask(ctx context.Context, task *taskqueue.TaskMessage) (bool, error) {
	ctx, span := tracing.Start(ctx, "worker.task.submit")
	span.SetAttributes(attribute.String("id", task.ID))
	defer span.End()

	err := w.pool.Submit(ctx, time.Second, task)
	if err != nil {
		if e := w.services.Task.ScheduleTask(ctx, task.ID, task.ScheduledAt); e != nil {
			w.log.Warnf("failed to update task %s scheduled_at to %d: %v", task.ID, task.ScheduledAt.UnixMilli(), e)
		}
		switch {
		case errors.Is(err, pool.ErrPoolTernimated):
			return false, errors.New("pool ternimated")
		case errors.Is(err, pool.ErrTimeout):
			w.log.Warnf("pool is busy")
			return false, nil
		}
	}
	return true, nil
}

func (w *Worker) runTask(ctx context.Context, task *taskqueue.TaskMessage) {
	ctx, span := tracing.Start(ctx, "worker.task.run",
		trace.WithAttributes(attribute.String("id", task.ID)))
	defer span.End()

	processing.Add(1)
	defer processing.Add(-1)

	task.Data = &taskqueue.MessageData{}
	err := task.UnmarshalData(task.Data)
	if err != nil {
		w.log.Errorf("failed to unmarshal task: %v", err)
		_ = w.services.Task.DeleteTask(ctx, task)
		return
	}

	err = w.handleTask(ctx, task)
	if err != nil {
		if errors.Is(ErrRateLimitExceeded, err) {
			return
		}
		// TODO: delete task when causes error too many times (maxReceiveCount)
		w.log.Errorf("failed to handle task: %v", err)
		return
	}
	_ = w.services.Task.DeleteTask(ctx, task)
}

func (w *Worker) registerEventHandler(bus eventbus.EventBus) {
	rs := redsync.New(goredis.NewPool(w.opts.RedisClient))
	bus.ClusteringSubscribe(eventbus.EventEventFanout, func(data []byte) {
		eventData := &eventbus.EventFanoutData{}
		if err := json.Unmarshal(data, eventData); err != nil {
			w.log.Errorf("failed to unmarshal event: %s", err)
			return
		}
		bus.Broadcast(context.TODO(), eventbus.EventEventFanout, eventData)
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

		w.services.Task.ScheduleAttempts(ctx, attempts)
	})
}

func (w *Worker) run() {
	options := &taskqueue.GetOptions{Count: 20}

	fetch := func(ctx context.Context) bool {
		ctx, span := tracing.Start(ctx, "worker.fetch")
		defer span.End()

		tasks, err := w.services.Task.GetTasks(ctx, options)
		if err != nil {
			w.log.Errorf("failed to fetch task: %v", err)
			return false
		}
		if len(tasks) == 0 {
			return false
		}

		continued := true
		for _, task := range tasks {
			_, err := w.submitTask(ctx, task)
			if err != nil {
				continued = false
			}
		}
		return continued
	}

	drain := func() {
		ctx, span := tracing.Start(context.Background(), "worker.drain")
		defer span.End()
		for {
			continued := fetch(ctx)
			if !continued {
				return
			}
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.services.Task.NotificationChannel():
			drain()
		case <-ticker.C:
			drain()
		}
	}
}

// Start starts worker
func (w *Worker) Start() error {
	w.log.Infow("starting worker", zap.Any("pool", map[string]interface{}{
		"size":      w.opts.PoolSize,
		"consumers": w.opts.PoolConcurrency,
	}))

	httpDeliverer := deliverer.NewHTTPDeliverer(w.opts.DelivererOptions)
	err := httpDeliverer.Setup()
	if err != nil {
		return err
	}
	w.deliverer = httpDeliverer

	for range runtime.NumCPU() {
		w.queueRequestLog.Consume(w.consumeQueue)
	}

	go w.run()

	w.services.Scheduler.AddTask(&schedule.Task{
		Name:     "worker.requeue",
		Interval: time.Minute,
		Do:       w.loadPending,
	})
	return nil
}

// Stop stops worker
func (w *Worker) Stop(ctx context.Context) error {
	w.log.Info("exiting")
	w.cancel()
	w.log.Named("pool").Infow("closing pool", "handling", processing.Load())
	w.pool.Shutdown()
	w.log.Named("pool").Info("closed pool")
	w.queueRequestLog.Close()
	w.log.Info("exit")

	return nil
}

func (w *Worker) consumeQueue(ctx context.Context, list []*entities.AttemptDetail) {
	err := w.db.AttemptDetails.BatchInsert(ctx, list)
	if err != nil {
		w.log.Errorf("failed to batch insert: %v", err)
	}
}

func (w *Worker) loadPending() {
	batchSize := w.opts.RequeueJobBatch

	var done bool
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
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
					w.services.Task.ScheduleAttempts(ctx, attempts)
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
		}

		if done {
			break
		}
	}
}

func (w *Worker) handleTask(ctx context.Context, task *taskqueue.TaskMessage) error {
	data := task.Data.(*taskqueue.MessageData)

	// validate endpoint
	cacheKey := constants.EndpointCacheKey.Build(data.EndpointId)
	endpoint, err := mcache.Load(ctx, cacheKey, nil, w.db.Endpoints.Get, data.EndpointId)
	if err != nil {
		return err
	}

	if err := w.validateEndpoint(ctx, task, endpoint); err != nil {
		if errors.Is(err, ErrTerminated) {
			return nil
		}
		return err
	}

	r, err := newRequestFromEndpoint(endpoint)
	if err != nil {
		// TODO: optimize error
		if err := w.db.Attempts.UpdateErrorCode(
			ctx, task.ID,
			entities.AttemptStatusCanceled,
			entities.AttemptErrorCodeUnknown,
		); err != nil {
			return err
		}
		return nil
	}

	iterator := plugins.LoadIterator()
	c := plugin.NewContext(ctx, r, nil)
	c.SetRequestBody([]byte(data.Event))
	for p := range iterator.Iterate(ctx, plugins.PhaseOutbound, endpoint.ID) {
		err = p.ExecuteOutbound(c)
		if err != nil {
			return fmt.Errorf("failed to execute %s plugin: %v", p.Name(), err)
		}
	}

	c.Request.Header.Set("Webhookx-Event-Id", data.EventID)
	c.Request.Header.Set("Webhookx-Delivery-Id", task.ID)

	request := &deliverer.Request{
		Request: c.Request,
		Body:    c.GetRequestBody(),
		Timeout: time.Duration(endpoint.Request.Timeout) * time.Millisecond,
	}

	// deliver the request
	startAt := time.Now()
	response := w.deliverer.Send(ctx, request)
	finishAt := time.Now()

	if response.Error != nil {
		w.log.Infof("failed to delivery event: %v", response.Error)
	}
	w.log.Debugf("delivery response: %v", response)

	result := buildAttemptResult(request, response)
	result.ID = task.ID
	result.AttemptedAt = types.NewTime(startAt)
	result.Exhausted = data.Attempt >= len(endpoint.Retry.Config.Attempts)
	if response.ACL.Denied {
		result.Exhausted = true
	}

	counter.Add(1)
	if result.Status == entities.AttemptStatusFailure {
		failures.Add(1)
	}

	if w.services.Metrics.Enabled {
		w.services.Metrics.AttemptTotalCounter.Add(1)
		if result.Status == entities.AttemptStatusFailure {
			w.services.Metrics.AttemptFailedCounter.Add(1)
		}
		w.services.Metrics.AttemptResponseDurationHistogram.Observe(response.Latancy.Seconds())
	}

	err = w.db.Attempts.UpdateDelivery(ctx, result)
	if err != nil {
		return err
	}

	ad := newAttemptDetail(task.ID, endpoint.WorkspaceId, response)
	w.queueRequestLog.Add(ctx, ad)

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

	w.services.Task.ScheduleAttempts(ctx, []*entities.Attempt{nextAttempt})
	return nil
}

func (w *Worker) validateEndpoint(ctx context.Context, task *taskqueue.TaskMessage, endpoint *entities.Endpoint) error {
	if endpoint == nil {
		if err := w.db.Attempts.UpdateErrorCode(
			ctx, task.ID,
			entities.AttemptStatusCanceled,
			entities.AttemptErrorCodeEndpointNotFound,
		); err != nil {
			return err
		}
		return ErrTerminated
	}

	if !endpoint.Enabled {
		if err := w.db.Attempts.UpdateErrorCode(
			ctx, task.ID,
			entities.AttemptStatusCanceled,
			entities.AttemptErrorCodeEndpointDisabled,
		); err != nil {
			return err
		}
		return ErrTerminated
	}
	if endpoint.RateLimit != nil {
		d := time.Duration(endpoint.RateLimit.Period) * time.Second
		res, err := w.services.RateLimiter.Allow(ctx, endpoint.ID, endpoint.RateLimit.Quota, d)
		if err != nil {
			return err
		}
		if !res.Allowed {
			task.ScheduledAt = time.Now().Add(d)
			w.log.Debugw("rate limit exceeded", "endpoint", endpoint.ID, "task", task.ID, "next", task.ScheduledAt)
			err := w.services.Task.ScheduleTask(ctx, task.ID, task.ScheduledAt)
			if err != nil {
				return err
			}
			return ErrRateLimitExceeded
		}
	}
	return nil
}

func newRequestFromEndpoint(endpoint *entities.Endpoint) (*http.Request, error) {
	r, err := http.NewRequest(endpoint.Request.Method, endpoint.Request.URL, nil)
	if err != nil {
		return nil, err
	}
	for _, header := range constants.DefaultDelivererRequestHeaders {
		r.Header.Add(header.Name, header.Value)
	}
	for k, v := range endpoint.Request.Headers {
		r.Header.Add(k, v)
	}
	return r, nil
}

func buildAttemptResult(request *deliverer.Request, response *deliverer.Response) *dao.AttemptResult {
	result := &dao.AttemptResult{
		Request: &entities.AttemptRequest{
			URL:    request.Request.URL.String(),
			Method: request.Request.Method,
		},
		Status: entities.AttemptStatusSuccess,
	}

	if response.Error != nil {
		if errors.Is(response.Error, context.DeadlineExceeded) {
			result.ErrorCode = utils.Pointer(entities.AttemptErrorCodeTimeout)
		} else if response.ACL.Denied {
			result.ErrorCode = utils.Pointer(entities.AttemptErrorCodeDenied)
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

func newAttemptDetail(id string, wid string, response *deliverer.Response) *entities.AttemptDetail {
	ad := &entities.AttemptDetail{}
	ad.ID = id
	ad.WorkspaceId = wid
	ad.RequestHeaders = utils.HeaderMap(response.Request.Request.Header)
	ad.RequestBody = utils.Pointer(string(response.Request.Body))
	if len(response.Header) > 0 {
		ad.ResponseHeaders = utils.Pointer(entities.Headers(utils.HeaderMap(response.Header)))
	}
	if response.ResponseBody != nil {
		ad.ResponseBody = utils.Pointer(string(response.ResponseBody))
	}
	return ad
}

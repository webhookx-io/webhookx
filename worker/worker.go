package worker

import (
	"context"
	"errors"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	plugintypes "github.com/webhookx-io/webhookx/pkg/plugin/types"
	"github.com/webhookx-io/webhookx/pkg/pool"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"runtime"
	"time"

	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.uber.org/zap"
)

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc

	opts WorkerOptions

	log *zap.SugaredLogger

	queue     taskqueue.TaskQueue
	deliverer deliverer.Deliverer
	DB        *db.DB
	pool      *pool.Pool
}

type WorkerOptions struct {
	RequeueJobBatch    int
	RequeueJobInterval time.Duration
	PoolSize           int
	PoolConcurrency    int
}

func NewWorker(opts WorkerOptions, db *db.DB, deliverer deliverer.Deliverer, queue taskqueue.TaskQueue) *Worker {
	opts.RequeueJobBatch = utils.DefaultIfZero(opts.RequeueJobBatch, constants.RequeueBatch)
	opts.RequeueJobInterval = utils.DefaultIfZero(opts.RequeueJobInterval, constants.RequeueInterval)
	opts.PoolSize = utils.DefaultIfZero(opts.PoolSize, 10000)
	opts.PoolConcurrency = utils.DefaultIfZero(opts.PoolConcurrency, runtime.NumCPU()*100)
	worker := &Worker{
		opts:      opts,
		queue:     queue,
		log:       zap.S(),
		deliverer: deliverer,
		DB:        db,
		pool:      pool.NewPool(opts.PoolSize, opts.PoolConcurrency),
	}

	return worker
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
				ctx := w.ctx
				tasks, err := w.queue.Get(ctx, options)
				if err != nil {
					w.log.Errorf("[worker] failed to get tasks from queue: %v", err)
					break
				}
				if len(tasks) == 0 {
					break
				}

				w.log.Debugf("[worker] receive tasks: %d", len(tasks))
				var errs []error
				for _, v := range tasks {
					task := v
					err = w.pool.SubmitFn(time.Second*5, func() {
						task.Data = &model.MessageData{}
						err = task.UnmarshalData(task.Data)
						if err != nil {
							w.log.Errorf("[worker] failed to unmarshal task: %v", err)
							_ = w.queue.Delete(ctx, task)
							return
						}

						err = w.handleTask(ctx, task)
						if err != nil {
							// TODO: delete task when causes error too many times (maxReceiveCount)
							w.log.Errorf("[worker] failed to handle task: %v", err)
							return
						}

						_ = w.queue.Delete(ctx, task)
					})
					if err != nil {
						if errors.Is(err, pool.ErrPoolTernimated) {
							return // worker is shutting down
						}
						errs = append(errs, err)
					}
				}
				if len(errs) > 0 {
					// FIXME: optmize log
					w.log.Warnf("[worker] failed to submit task to pool: %v", errs) // consider tuning pool configuration
					break
				}
			}
		}
	}
}

// Start starts worker
func (w *Worker) Start() error {
	go w.run()

	w.ctx, w.cancel = context.WithCancel(context.Background())
	schedule.Schedule(w.ctx, w.processRequeue, w.opts.RequeueJobInterval)
	w.log.Infof("[worker] created pool(size=%d, concurrency=%d)", w.opts.PoolSize, w.opts.PoolConcurrency)
	w.log.Info("[worker] started")
	return nil
}

// Stop stops worker
func (w *Worker) Stop() error {
	w.cancel()
	w.log.Info("[worker] goroutine pool is shutting down")
	w.pool.Shutdown()
	w.log.Info("[worker] stopped")

	return nil
}

func (w *Worker) processRequeue() {
	batch := w.opts.RequeueJobBatch
	ctx := context.Background()
	for {
		attempts, err := w.DB.Attempts.ListUnqueued(ctx, batch)
		if err != nil {
			w.log.Errorf("failed to query unqueued attempts: %v", err)
			break
		}
		if len(attempts) == 0 {
			break
		}

		tasks := make([]*taskqueue.TaskMessage, 0, len(attempts))
		for _, attempt := range attempts {
			task := &taskqueue.TaskMessage{
				ID:          attempt.ID,
				ScheduledAt: attempt.ScheduledAt.Time,
				Data: &model.MessageData{
					EventID:    attempt.EventId,
					EndpointId: attempt.EndpointId,
					Attempt:    attempt.AttemptNumber,
				},
			}
			tasks = append(tasks, task)
		}

		for _, task := range tasks {
			err := w.queue.Add(ctx, []*taskqueue.TaskMessage{task})
			if err != nil {
				w.log.Warnf("failed to add task to queue: %v", err)
				continue
			}
			err = w.DB.Attempts.UpdateStatus(ctx, task.ID, entities.AttemptStatusQueued)
			if err != nil {
				w.log.Warnf("failed to update attempt status: %v", err)
			}
		}

		if len(attempts) < batch {
			break
		}
	}

}

func (w *Worker) handleTask(ctx context.Context, task *taskqueue.TaskMessage) error {
	data := task.Data.(*model.MessageData)

	// verify endpoint
	endpoint, err := w.DB.Endpoints.Get(ctx, data.EndpointId)
	if err != nil {
		return err
	}
	if endpoint == nil {
		return w.DB.Attempts.UpdateErrorCode(ctx, task.ID, entities.AttemptStatusCanceled, entities.AttemptErrorCodeEndpointNotFound)
	}
	if !endpoint.Enabled {
		return w.DB.Attempts.UpdateErrorCode(ctx, task.ID, entities.AttemptStatusCanceled, entities.AttemptErrorCodeEndpointDisabled)
	}

	// verify event
	event, err := w.DB.Events.Get(ctx, data.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		return w.DB.Attempts.UpdateErrorCode(ctx, task.ID, entities.AttemptStatusCanceled, entities.AttemptErrorCodeUnknown)
	}

	plugins, err := w.DB.Plugins.ListEndpointPlugin(ctx, endpoint.ID)
	if err != nil {
		return err
	}

	workspace, err := w.DB.Workspaces.Get(ctx, endpoint.WorkspaceId)
	if err != nil {
		return err
	}

	pluginReq := plugintypes.Request{
		URL:     endpoint.Request.URL,
		Method:  endpoint.Request.Method,
		Headers: endpoint.Request.Headers,
		Payload: event.Data,
	}
	if pluginReq.Headers == nil {
		pluginReq.Headers = make(map[string]string)
	}
	pluginCtx := &plugintypes.Context{
		Workspace: workspace,
	}
	for _, p := range plugins {
		err = plugin.ExecutePlugin(p, &pluginReq, pluginCtx)
		if err != nil {
			return err
		}
	}

	request := &deliverer.Request{
		Request: nil,
		URL:     pluginReq.URL,
		Method:  pluginReq.Method,
		Payload: pluginReq.Payload,
		Headers: pluginReq.Headers,
		Timeout: time.Duration(endpoint.Request.Timeout) * time.Millisecond,
	}

	// deliver the request
	startAt := time.Now()
	response := w.deliverer.Deliver(request)
	finishAt := time.Now()

	if response.Error != nil {
		w.log.Infof("[worker] failed to delivery event: %v", response.Error)
	}
	w.log.Debugf("[worker] delivery response: %v", response)

	result := buildAttemptResult(request, response)
	result.AttemptedAt = types.NewTime(startAt)
	result.Exhausted = data.Attempt >= len(endpoint.Retry.Config.Attempts)

	err = w.DB.Attempts.UpdateDelivery(ctx, task.ID, result)
	if err != nil {
		return err
	}

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
	err = w.DB.AttemptDetails.Upsert(ctx, attemptDetail)
	if err != nil {
		return err
	}

	if result.Status == entities.AttemptStatusSuccess {
		return nil
	}

	if result.Exhausted {
		w.log.Debugf("[worker] webhook delivery exhausted : %s", task.ID)
		return nil
	}

	delay := endpoint.Retry.Config.Attempts[data.Attempt]
	nextAttempt := &entities.Attempt{
		ID:            utils.KSUID(),
		EventId:       event.ID,
		EndpointId:    endpoint.ID,
		Status:        entities.AttemptStatusInit,
		AttemptNumber: data.Attempt + 1,
		ScheduledAt:   types.NewTime(finishAt.Add(time.Second * time.Duration(delay))),
		TriggerMode:   entities.AttemptTriggerModeAutomatic,
	}
	nextAttempt.WorkspaceId = endpoint.WorkspaceId

	err = w.DB.Attempts.Insert(ctx, nextAttempt)
	if err != nil {
		return err
	}

	task = &taskqueue.TaskMessage{
		ID:          nextAttempt.ID,
		ScheduledAt: nextAttempt.ScheduledAt.Time,
		Data: &model.MessageData{
			EventID:    data.EventID,
			EndpointId: data.EndpointId,
			Attempt:    nextAttempt.AttemptNumber,
		},
	}

	err = w.queue.Add(ctx, []*taskqueue.TaskMessage{task})
	if err != nil {
		w.log.Warnf("[worker] failed to add task to queue: %v", err)
	}
	err = w.DB.Attempts.UpdateStatus(ctx, nextAttempt.ID, entities.AttemptStatusQueued)
	if err != nil {
		w.log.Warnf("[worker] failed to update attempt status: %v", err)
	}
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

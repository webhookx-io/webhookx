package worker

import (
	"context"
	"errors"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/safe"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	ErrServerStarted = errors.New("already started")
	ErrServerStopped = errors.New("already stopped")
)

type Worker struct {
	ctx     context.Context
	mux     sync.Mutex
	started bool
	log     *zap.SugaredLogger

	queue     queue.TaskQueue
	deliverer deliverer.Deliverer
	DB        *db.DB
}

func NewWorker(ctx context.Context, cfg *config.WorkerConfig, db *db.DB, queue queue.TaskQueue) *Worker {
	worker := &Worker{
		ctx:       ctx,
		queue:     queue,
		log:       zap.S(),
		deliverer: deliverer.NewHTTPDeliverer(&cfg.Deliverer),
		DB:        db,
	}

	return worker
}

func (w *Worker) run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			w.log.Info("[worker] receive stop signal")
			return
		case <-ticker.C:
			// w.log.Debugf("[worker] ticker tick")
			for {
				task, err := w.queue.Get()
				if err != nil {
					w.log.Errorf("[worker] failed to get task from queue: %v", err)
					break
				}
				if task == nil {
					break
				}
				w.log.Debugf("[worker] receive task: %v", task)
				safe.Go(func() {
					task.Data = &model.MessageData{}
					err = task.UnmarshalData(task.Data)
					if err != nil {
						w.log.Errorf("[worker] failed to unmarshal task: %v", err)
						w.queue.Delete(task)
						return
					}

					err = w.handleTask(context.Background(), task)
					if err != nil {
						// TODO: delete task when causes error too many times (maxReceiveCount)
						w.log.Errorf("[worker] failed to handle task: %v", err)
						return
					}

					w.queue.Delete(task)
				})
			}
		}
	}
}

// Start starts worker
func (w *Worker) Start() error {
	w.mux.Lock()
	defer w.mux.Unlock()

	if w.started {
		return ErrServerStarted
	}

	go w.run()
	w.started = true
	w.log.Info("[worker] started")

	return nil
}

// Stop stops worker
func (w *Worker) Stop() error {
	w.mux.Lock()
	defer w.mux.Unlock()

	if !w.started {
		return ErrServerStopped
	}

	// TODO: wait for all go routines finished
	time.Sleep(time.Second)

	w.started = false
	w.log.Info("[worker] stopped")

	return nil
}

func (w *Worker) handleTask(ctx context.Context, task *queue.TaskMessage) error {
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

	request := &deliverer.Request{
		URL:     endpoint.Request.URL,
		Method:  endpoint.Request.Method,
		Payload: event.Data,
		Headers: endpoint.Request.Headers,
		Timeout: time.Duration(endpoint.Request.Timeout) * time.Millisecond,
	}

	result := &dao.DeliveryResult{
		AttemptAt: time.Now(),
	}

	response := w.deliverer.Deliver(request)
	if response.Error != nil {
		if errors.Is(response.Error, context.DeadlineExceeded) {
			result.ErrorCode = utils.Pointer(entities.AttemptErrorCodeTimeout)
		} else {
			result.ErrorCode = utils.Pointer(entities.AttemptErrorCodeUnknown)
		}
		w.log.Infof("[worker] failed to delivery event: %v", response.Error)
	}

	w.log.Debugf("[worker] delivery response: %v", response)

	if response.Is2xx() {
		result.Status = entities.AttemptStatusSuccess
	} else {
		result.Status = entities.AttemptStatusFailure
	}

	result.Request = &entities.AttemptRequest{
		URL:     request.URL,
		Method:  request.Method,
		Headers: utils.HeaderMap(request.Request.Header),
		Body:    utils.Pointer(string(request.Payload)),
	}
	if response.StatusCode != 0 {
		result.Response = &entities.AttemptResponse{
			Status:  response.StatusCode,
			Headers: utils.HeaderMap(response.Header),
			Body:    utils.Pointer(string(response.ResponseBody)),
		}
	}

	err = w.DB.Attempts.UpdateDelivery(ctx, task.ID, result)
	if err != nil {
		return err
	}

	if result.Status == entities.AttemptStatusSuccess {
		return nil
	}

	if data.Attempt >= len(endpoint.Retry.Config.Attempts) {
		w.log.Debugf("[worker] webhook delivery exhausted : %s", task.ID)
		return nil
	}

	NextAttempt := &entities.Attempt{
		ID:            utils.KSUID(),
		EventId:       event.ID,
		EndpointId:    endpoint.ID,
		Status:        entities.AttemptStatusInit,
		AttemptNumber: data.Attempt + 1,
	}
	NextAttempt.WorkspaceId = endpoint.WorkspaceId

	err = w.DB.AttemptsWS.Insert(ctx, NextAttempt)
	if err != nil {
		return err
	}

	task = &queue.TaskMessage{
		ID: NextAttempt.ID,
		Data: &model.MessageData{
			EventID:    data.EventID,
			EndpointId: data.EndpointId,
			Delay:      endpoint.Retry.Config.Attempts[NextAttempt.AttemptNumber-1],
			Attempt:    NextAttempt.AttemptNumber,
		},
	}
	err = w.queue.Add(task, utils.DurationS(task.Data.(*model.MessageData).Delay))
	if err != nil {
		w.log.Warnf("[worker] failed to add task to queue: %v", err)
	}
	err = w.DB.Attempts.UpdateStatus(ctx, NextAttempt.ID, entities.AttemptStatusQueued)
	if err != nil {
		w.log.Warnf("[worker] failed to update attempt status: %v", err)
	}
	return nil
}

package worker

import (
	"context"
	"errors"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
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

	// verifying endpoint
	endpoint, err := w.DB.Endpoints.Get(ctx, data.EndpointId)
	if err != nil {
		return err
	}
	if endpoint == nil {
		w.log.Warnf("endpoint not found: %s", data.EndpointId)
		return w.DB.Attempts.UpdateStatus(ctx, task.ID, entities.AttemptStatusCanceled)
	}
	if !endpoint.Enabled {
		w.log.Warnf("endpoint is disabled: %s", data.EndpointId)
		return w.DB.Attempts.UpdateStatus(ctx, task.ID, entities.AttemptStatusEndpointDisabled)
	}

	// verifying event
	event, err := w.DB.Events.Get(ctx, data.EventID)
	if err != nil {
		return err
	}
	if event == nil {
		w.log.Warnf("event not found: %s", data.EventID)
		return w.DB.Attempts.UpdateStatus(ctx, task.ID, entities.AttemptStatusCanceled)
	}

	request := &deliverer.Request{
		URL:     endpoint.Request.URL,
		Method:  endpoint.Request.Method,
		Payload: event.Data,
		//Headers: nil, TODO
		Timeout: time.Duration(endpoint.Request.Timeout) * time.Millisecond,
	}

	now := time.Now()
	response := w.deliverer.Deliver(request)
	if response.Error != nil {
		w.log.Infof("[worker] failed to send webhook %v", response.Error)
	}

	w.log.Debugf("[worker] webhook response: %v", response)

	attemptRequest := &entities.AttemptRequest{
		URL:    request.URL,
		Method: request.Method,
		Header: map[string]string{},
		Body:   string(request.Payload),
	}

	attemptResponse := &entities.AttemptResponse{
		Status: response.StatusCode,
		Header: map[string]string{},
		Body:   string(response.ResponseBody),
	}

	var status entities.AttemptStatus
	if response.Is2xx() {
		//w.log.Debugf("[worker] deliver webhook successful")
		status = entities.AttemptStatusSuccess
	} else {
		//w.log.Debugf("[worker] deliver webhook failed")
		status = entities.AttemptStatusFailure
	}

	err = w.DB.Attempts.UpdateDelivery(ctx, task.ID, attemptRequest, attemptResponse, now, status)
	if err != nil {
		return err
	}

	if status == entities.AttemptStatusSuccess {
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

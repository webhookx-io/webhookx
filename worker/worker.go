package worker

import (
	"context"
	"errors"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/safe"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.uber.org/zap"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	ErrServerStarted = errors.New("already started")
	ErrServerStopped = errors.New("already stopped")
)

type Worker struct {
	mux     sync.Mutex
	started bool
	signal  chan struct{}
	log     *zap.SugaredLogger

	queue queue.TaskQueue

	DB *db.DB
}

func NewWorker(cfg *config.Config, db *db.DB, queue queue.TaskQueue) *Worker {
	worker := &Worker{
		signal: make(chan struct{}),
		queue:  queue,
		log:    zap.S(),
		DB:     db,
	}

	return worker
}

func (w *Worker) run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.signal:
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

	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		w.log.Infof("[worker] worker is shutting down")
		w.Stop()
	}()

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

	w.signal <- struct{}{}

	// TODO: wait for all go routines finished
	time.Sleep(time.Second)

	w.started = false
	w.log.Info("[worker] stopped")

	return nil
}

func (w *Worker) handleTask(ctx context.Context, task *queue.TaskMessage) error {
	data := task.Data.(*model.MessageData)

	event, err := w.DB.Events.Get(ctx, data.EventID)
	if err != nil {
		return err
	} else if event == nil {
		w.log.Warnf("event not found: %s", data.EventID)
		return nil
	}

	endpoint, err := w.DB.Endpoints.Get(ctx, data.EndpointId)
	if err != nil {
		return err
	} else if endpoint == nil {
		w.log.Warnf("endpoint not found: %s", data.EndpointId)
		return nil
	}

	client := &http.Client{
		// TODO: timeout
	}
	httpd := deliverer.NewHTTPDeliverer(client)

	request := &deliverer.Request{
		URL:     endpoint.Request.URL,
		Method:  endpoint.Request.Method,
		Payload: event.Data,
		//Headers: nil, TODO
	}
	response, err := httpd.Deliver(request)
	if err != nil {
		w.log.Infof("[worker] failed to send webhook %v", err)
	}

	w.log.Debugf("[worker] webhook response: %v", response)

	// TODO: add http log record

	if response.Is2xx() {
		w.log.Debugf("[worker] deliver webhook successful")
		return nil
	}

	if data.AttemptLeft == 0 {
		w.log.Debugf("[worker] webhook delivery exhausted")
		return nil
	}

	task = &queue.TaskMessage{
		ID: utils.UUID(),
		Data: &model.MessageData{
			EventID:     data.EventID,
			EndpointId:  data.EndpointId,
			Time:        0,
			Attempt:     data.Attempt + 1,
			Delay:       endpoint.Retry.Config.Attempts[data.Attempt],
			AttemptLeft: data.AttemptLeft - 1,
		},
	}
	return w.queue.Add(task, utils.DurationS(task.Data.(*model.MessageData).Delay))
}

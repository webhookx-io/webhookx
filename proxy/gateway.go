package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/queue/redis"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/proxy/middlewares"
	"github.com/webhookx-io/webhookx/proxy/router"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"net/http"
	"os"
	"runtime"
	"time"
)

var (
	ErrQueueDisabled = errors.New("queue is disabled")
)

type Gateway struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.ProxyConfig

	log    *zap.SugaredLogger
	s      *http.Server
	router *router.Router // TODO: happens-before
	db     *db.DB

	dispatcher *dispatcher.Dispatcher

	queue   queue.Queue
	metrics *metrics.Metrics
}

func NewGateway(cfg *config.ProxyConfig, db *db.DB, dispatcher *dispatcher.Dispatcher, metrics *metrics.Metrics) *Gateway {
	var q queue.Queue
	switch cfg.Queue.Type {
	case "redis":
		q, _ = redis.NewRedisQueue(redis.RedisQueueOptions{
			Client: cfg.Queue.Redis.GetClient(),
		}, zap.S(), metrics)
	}

	gw := &Gateway{
		cfg:        cfg,
		log:        zap.S(),
		router:     router.NewRouter(nil),
		db:         db,
		dispatcher: dispatcher,
		queue:      q,
		metrics:    metrics,
	}

	r := mux.NewRouter()
	r.Use(middlewares.PanicRecovery)
	if metrics.Enabled {
		r.Use(middlewares.NewMetricsMiddleware(metrics).Handle)
	}
	r.PathPrefix("/").HandlerFunc(gw.Handle)

	gw.s = &http.Server{
		Handler: r,
		Addr:    cfg.Listen,

		ReadTimeout:  time.Duration(cfg.TimeoutRead) * time.Second,
		WriteTimeout: time.Duration(cfg.TimeoutWrite) * time.Second,
	}

	return gw
}

func (gw *Gateway) buildRouter() {
	routes := make([]*router.Route, 0)
	sources, err := gw.db.Sources.List(context.TODO(), &query.SourceQuery{})
	if err != nil {
		gw.log.Warnf("[proxy] failed to build router: %v", err)
		return
	}
	for _, source := range sources {
		route := router.Route{
			Paths:   []string{source.Path},
			Methods: source.Methods,
			Handler: source,
		}
		routes = append(routes, &route)
	}
	gw.router = router.NewRouter(routes)
}

func (gw *Gateway) Handle(w http.ResponseWriter, r *http.Request) {
	source, _ := gw.router.Execute(r).(*entities.Source)
	if source == nil {
		exit(w, 404, `{"message": "not found"}`, nil)
		return
	}

	ctx := ucontext.WithContext(r.Context(), &ucontext.UContext{
		WorkspaceID: source.WorkspaceId,
	})
	r = r.WithContext(ctx)

	var event entities.Event
	r.Body = http.MaxBytesReader(w, r.Body, gw.cfg.MaxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			code := http.StatusRequestEntityTooLarge
			http.Error(w, http.StatusText(code), code)
			return
		}
		utils.JsonResponse(400, w, types.ErrorResponse{
			Message: err.Error(),
		})
		return
	}

	event.ID = utils.KSUID()
	event.IngestedAt = types.Time{Time: time.Now()}
	event.WorkspaceId = source.WorkspaceId
	if err := event.Validate(); err != nil {
		utils.JsonResponse(400, w, types.ErrorResponse{
			Message: "Request Validation",
			Error:   err,
		})
		return
	}

	err := gw.ingestEvent(r.Context(), source.Async, &event)
	if err != nil {
		gw.log.Errorf("[proxy] failed to ingest event: %v", err)
		exit(w, 500, `{"message": "internal error"}`, nil)
		return
	}
	if gw.metrics.Enabled {
		gw.metrics.EventTotalCounter.Add(1)
	}

	if source.Response != nil {
		exit(w, source.Response.Code, source.Response.Body, headers{"Content-Type": source.Response.ContentType})
		return
	}

	// default response
	exit(w, int(gw.cfg.Response.Code), gw.cfg.Response.Body, headers{"Content-Type": gw.cfg.Response.ContentType})
}

func (gw *Gateway) ingestEvent(ctx context.Context, async bool, event *entities.Event) error {
	if async {
		if gw.queue == nil {
			return ErrQueueDisabled
		}

		bytes, err := json.Marshal(event)
		if err != nil {
			return err
		}

		msg := queue.Message{
			Data:        bytes,
			Time:        time.Now(),
			WorkspaceID: event.WorkspaceId,
		}
		return gw.queue.Enqueue(ctx, &msg)
	}

	return gw.dispatcher.Dispatch(ctx, event)
}

// Start starts an HTTP server
func (gw *Gateway) Start() {
	gw.ctx, gw.cancel = context.WithCancel(context.Background())

	go func() {
		if err := gw.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("Failed to start Gateway : %v", err)
			os.Exit(1)
		}
	}()

	schedule.Schedule(gw.ctx, gw.buildRouter, time.Second)

	if gw.queue != nil {
		listeners := runtime.GOMAXPROCS(0)
		gw.log.Infof("[proxy] starting %d queue listener", listeners)
		for i := 0; i < listeners; i++ {
			go gw.listenQueue()
		}
	}

	gw.log.Info("[proxy] started")
}

// Stop stops the HTTP server
func (gw *Gateway) Stop() error {
	gw.cancel()

	if err := gw.s.Shutdown(context.TODO()); err != nil {
		// Error from closing listeners, or context timeout:
		return err
	}
	return nil
}

func (gw *Gateway) listenQueue() {
	opts := &queue.Options{
		Count:   20,
		Block:   true,
		Timeout: time.Second,
	}
	for {
		select {
		case <-gw.ctx.Done():
			return
		default:
			ctx := context.TODO()
			messages, err := gw.queue.Dequeue(ctx, opts)
			if err != nil {
				gw.log.Warnf("[proxy] [queue] failed to dequeue: %v", err)
				time.Sleep(time.Second)
				continue
			}
			if len(messages) == 0 {
				continue
			}

			events := make([]*entities.Event, 0, len(messages))
			for _, message := range messages {
				var event entities.Event
				err = json.Unmarshal(message.Data, &event)
				if err != nil {
					gw.log.Warnf("[proxy] [queue] faield to unmarshal message: %v", err)
					continue
				}
				event.WorkspaceId = message.WorkspaceID
				events = append(events, &event)
			}

			err = gw.dispatcher.DispatchBatch(ctx, events)
			if err != nil {
				gw.log.Warnf("[proxy] [queue] failed to dispatch event in batch: %v", err)
				continue
			}
			_ = gw.queue.Delete(ctx, messages)
		}
	}
}

type headers map[string]string

func exit(w http.ResponseWriter, status int, body string, headers headers) {
	for header, value := range constants.DefaultResponseHeaders {
		w.Header().Set(header, value)
	}

	if len(headers) > 0 {
		for header, value := range headers {
			w.Header().Set(header, value)
		}
	}

	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

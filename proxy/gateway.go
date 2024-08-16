package proxy

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/proxy/router"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

type Gateway struct {
	ctx context.Context

	cfg *config.ProxyConfig

	log    *zap.SugaredLogger
	s      *http.Server
	router *router.Router // TODO: happens-before
	db     *db.DB
	queue  queue.TaskQueue
}

func NewGateway(cfg *config.ProxyConfig, db *db.DB, queue queue.TaskQueue) *Gateway {
	gw := &Gateway{
		ctx:    context.Background(),
		cfg:    cfg,
		log:    zap.S(),
		router: router.NewRouter(nil),
		db:     db,
		queue:  queue,
	}

	r := mux.NewRouter()
	r.Use(panicRecovery)
	r.PathPrefix("/").HandlerFunc(gw.Handle)

	gw.s = &http.Server{
		Handler: r,
		Addr:    cfg.Listen,

		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}

	return gw
}

func (gw *Gateway) buildRouter() error {
	routes := make([]*router.Route, 0)
	sources, err := gw.db.Sources.List(context.TODO(), &query.SourceQuery{})
	if err != nil {
		return err
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
	return nil
}

func (gw *Gateway) routerRebuildLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-gw.ctx.Done():
			return
		case <-ticker.C:
			err := gw.buildRouter()
			if err != nil {
				gw.log.Warnf("[proxy] failed to build router: %v", err)
			}
		}
	}
}

func (gw *Gateway) DispatchEvent(ctx context.Context, event *entities.Event) error {
	endpoints, err := listSubscribedEndpoints(ctx, gw.db, event.EventType)
	if err != nil {
		return err
	}

	attempts := make([]*entities.Attempt, 0, len(endpoints))
	tasks := make([]*queue.TaskMessage, 0, len(endpoints))

	err = gw.db.TX(ctx, func(ctx context.Context) error {
		err := gw.db.Events.Insert(ctx, event)
		if err != nil {
			return err
		}

		for _, endpoint := range endpoints {
			attempt := &entities.Attempt{
				ID:            utils.UUID(),
				EventId:       event.ID,
				EndpointId:    endpoint.ID,
				Status:        entities.AttemptStatusInit,
				AttemptNumber: 1,
			}
			attempt.WorkspaceId = endpoint.WorkspaceId

			task := &queue.TaskMessage{
				ID: attempt.ID,
				Data: &model.MessageData{
					EventID:    event.ID,
					EndpointId: endpoint.ID,
					Delay:      endpoint.Retry.Config.Attempts[0],
					Attempt:    1,
				},
			}
			attempts = append(attempts, attempt)
			tasks = append(tasks, task)
		}

		return gw.db.AttemptsWS.BatchInsert(ctx, attempts)
	})
	if err != nil {
		return err
	}

	for _, task := range tasks {
		err := gw.queue.Add(task, utils.DurationS(task.Data.(*model.MessageData).Delay))
		if err != nil {
			gw.log.Warnf("failed to add task to queue: %v", err)
		}
		err = gw.db.AttemptsWS.UpdateStatus(ctx, task.ID, entities.AttemptStatusQueued)
		if err != nil {
			gw.log.Warnf("failed to update attempt status: %v", err)
		}
	}

	return nil
}

func listSubscribedEndpoints(ctx context.Context, db *db.DB, eventType string) (list []*entities.Endpoint, err error) {
	var q query.EndpointQuery
	endpoints, err := db.EndpointsWS.List(ctx, &q)
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

func (gw *Gateway) Handle(w http.ResponseWriter, r *http.Request) {
	source, _ := gw.router.Execute(r).(*entities.Source)
	if source == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		w.Write([]byte(`{"message": "not found"}`))
		return
	}

	ctx := ucontext.WithContext(r.Context(), &ucontext.UContext{
		WorkspaceID: source.WorkspaceId,
	})
	r = r.WithContext(ctx)

	var event entities.Event
	event.ID = utils.UUID()
	r.Body = http.MaxBytesReader(w, r.Body, gw.cfg.MaxRequestBodySize)
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			code := http.StatusRequestEntityTooLarge
			http.Error(w, http.StatusText(code), code)
			return
		}
		utils.JsonResponse(400, w, ErrorResponse{
			Message: err.Error(),
		})
		return
	}

	if err := event.Validate(); err != nil {
		utils.JsonResponse(400, w, ErrorResponse{
			Message: "Reqeust Validation",
			Error:   err,
		})
		return
	}
	event.WorkspaceId = source.WorkspaceId
	err := gw.DispatchEvent(r.Context(), &event)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write([]byte(`{"message": "internal error"}`))
		return
	}

	if source.Response != nil {
		w.Header().Set("Content-Type", source.Response.ContentType)
		w.WriteHeader(source.Response.Code)
		w.Write([]byte(source.Response.Body))
		return
	}

	w.Header().Set("Content-Type", gw.cfg.Response.ContentType)
	w.WriteHeader(int(gw.cfg.Response.Code))
	w.Write([]byte(gw.cfg.Response.Body))
}

// Start starts an HTTP server
func (gw *Gateway) Start() {
	go func() {
		if err := gw.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("Failed to start Gateway : %v", err)
			os.Exit(1)
		}
	}()

	go gw.routerRebuildLoop()

	gw.log.Info("[proxy] started")
}

// Stop stops the HTTP server
func (gw *Gateway) Stop() error {
	if err := gw.s.Shutdown(context.TODO()); err != nil {
		// Error from closing listeners, or context timeout:
		return err
	}
	return nil
}

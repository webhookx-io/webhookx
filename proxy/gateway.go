package proxy

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/proxy/router"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

type Gateway struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.ProxyConfig

	log    *zap.SugaredLogger
	s      *http.Server
	router *router.Router // TODO: happens-before
	db     *db.DB

	dispatcher dispatcher.Dispatcher
}

func NewGateway(cfg *config.ProxyConfig, db *db.DB, dispatcher dispatcher.Dispatcher) *Gateway {
	gw := &Gateway{
		cfg:        cfg,
		log:        zap.S(),
		router:     router.NewRouter(nil),
		db:         db,
		dispatcher: dispatcher,
	}

	r := mux.NewRouter()
	r.Use(panicRecovery)
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
	event.ID = utils.KSUID()
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
			Message: "Request Validation",
			Error:   err,
		})
		return
	}
	event.WorkspaceId = source.WorkspaceId
	err := gw.dispatcher.Dispatch(r.Context(), &event)
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
	gw.ctx, gw.cancel = context.WithCancel(context.Background())

	go func() {
		if err := gw.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.S().Errorf("Failed to start Gateway : %v", err)
			os.Exit(1)
		}
	}()

	schedule.Schedule(gw.ctx, gw.buildRouter, time.Second)

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

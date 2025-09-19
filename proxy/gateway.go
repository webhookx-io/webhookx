package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/loglimiter"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/queue/redis"
	"github.com/webhookx-io/webhookx/pkg/ratelimiter"
	"github.com/webhookx-io/webhookx/pkg/schedule"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/store"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/proxy/middlewares"
	"github.com/webhookx-io/webhookx/proxy/router"
	"github.com/webhookx-io/webhookx/service"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	ErrQueueDisabled = errors.New("queue is disabled")
)

var (
	counter  atomic.Int64
	failures atomic.Int64
)

type Gateway struct {
	ctx    context.Context
	cancel context.CancelFunc

	cfg *config.ProxyConfig

	log *zap.SugaredLogger
	s   *http.Server

	router        atomic.Value
	routerVersion string

	db *db.DB

	dispatcher *dispatcher.Dispatcher

	queue   queue.Queue
	metrics *metrics.Metrics
	tracer  *tracing.Tracer
	bus     eventbus.Bus

	limiter     *loglimiter.Limiter
	srv         *service.Service
	rateLimiter ratelimiter.RateLimiter
}

type Options struct {
	Cfg         *config.ProxyConfig
	Middlewares []mux.MiddlewareFunc
	DB          *db.DB
	Dispatcher  *dispatcher.Dispatcher
	Metrics     *metrics.Metrics
	Tracer      *tracing.Tracer
	EventBus    eventbus.Bus
	Srv         *service.Service
	RateLimiter ratelimiter.RateLimiter
}

func init() {
	stats.Register(stats.ProviderFunc(func() map[string]interface{} {
		return map[string]interface{}{
			"gateway.requests":        counter.Load(),
			"gateway.failed_requests": failures.Load(),
		}
	}))
}

func NewGateway(opts Options) *Gateway {
	var q queue.Queue
	switch opts.Cfg.Queue.Type {
	case "redis":
		q, _ = redis.NewRedisQueue(redis.Options{
			StreamName:        constants.QueueRedisQueueName,
			ConsumerGroupName: constants.QueueRedisGroupName,
			ConsumerName:      constants.QueueRedisConsumerName,
			VisibilityTimeout: constants.QueueRedisVisibilityTimeout,
			Listeners:         runtime.GOMAXPROCS(0),
			Client:            opts.Cfg.Queue.Redis.GetClient(),
		}, zap.S())
		stats.Register(q)
	}

	gw := &Gateway{
		cfg:         opts.Cfg,
		log:         zap.S().Named("proxy"),
		db:          opts.DB,
		dispatcher:  opts.Dispatcher,
		queue:       q,
		metrics:     opts.Metrics,
		tracer:      opts.Tracer,
		bus:         opts.EventBus,
		limiter:     loglimiter.NewLimiter(time.Second),
		srv:         opts.Srv,
		rateLimiter: opts.RateLimiter,
	}

	gw.router.Store(router.NewRouter(nil))

	r := mux.NewRouter()
	r.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
			counter.Add(1)
		})
	})

	for _, m := range opts.Middlewares {
		r.Use(m)
	}
	r.Use(middlewares.PanicRecovery)
	r.PathPrefix("/").HandlerFunc(gw.Handle)

	gw.s = &http.Server{
		Handler: r,
		Addr:    gw.cfg.Listen,

		ReadTimeout:  time.Duration(gw.cfg.TimeoutRead) * time.Second,
		WriteTimeout: time.Duration(gw.cfg.TimeoutWrite) * time.Second,
	}

	return gw
}

func (gw *Gateway) buildRouter(version string) {
	gw.log.Debugw("building router", "version", version)

	sources, err := gw.db.Sources.List(context.TODO(), &query.SourceQuery{})
	if err != nil {
		gw.log.Warnf("failed to build router: %v", err)
		return
	}

	routes := make([]*router.Route, 0)
	for _, source := range sources {
		route := router.Route{
			Paths:   []string{source.Path},
			Methods: source.Methods,
			Handler: source,
		}
		routes = append(routes, &route)
	}
	gw.router.Store(router.NewRouter(routes))
	gw.routerVersion = version
}

func (gw *Gateway) Handle(w http.ResponseWriter, r *http.Request) {
	ok := gw.handle(w, r)
	if !ok {
		failures.Add(1)
	}
}

func (gw *Gateway) handle(w http.ResponseWriter, r *http.Request) bool {
	router := gw.router.Load().(*router.Router)
	source, _ := router.Execute(r).(*entities.Source)
	if source == nil {
		response.JSON(w, 404, types.ErrorResponse{Message: "not found"})
		return false
	}

	ctx := ucontext.WithContext(context.WithoutCancel(r.Context()), &ucontext.UContext{
		WorkspaceID: source.WorkspaceId,
	})

	if gw.tracer != nil {
		tracingCtx, span := gw.tracer.Start(ctx, "proxy.handle", trace.WithSpanKind(trace.SpanKindServer))
		span.SetAttributes(attribute.String("source.id", source.ID))
		span.SetAttributes(attribute.String("source.name", utils.PointerValue(source.Name)))
		span.SetAttributes(attribute.String("source.workspace_id", source.WorkspaceId))
		span.SetAttributes(attribute.Bool("source.async", source.Async))
		span.SetAttributes(semconv.HTTPRoute(source.Path))
		defer span.End()
		ctx = tracingCtx
	}

	if source.RateLimit != nil {
		d := time.Duration(source.RateLimit.Period) * time.Second
		res, err := gw.rateLimiter.Allow(ctx, source.ID, source.RateLimit.Quota, d)
		if err != nil {
			gw.log.Errorf("failed to execute rate limiting: %v", err)
			response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
			return false
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(source.RateLimit.Quota))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(math.Ceil(res.Reset.Seconds()))))
		if !res.Allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(res.RetryAfter.Seconds()))))
			gw.log.Debugw("rate limit exceeded", "source", source.ID)
			response.JSON(w, 429, types.ErrorResponse{Message: "rate limit exceeded"})
			return false
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, gw.cfg.MaxRequestBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			code := http.StatusRequestEntityTooLarge
			http.Error(w, http.StatusText(code), code)
			return false
		}
	}

	plugins, err := listSourcePlugins(ctx, gw.db, source.ID)
	if err != nil {
		response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
		return false
	}

	for _, p := range plugins {
		executor, err := p.Plugin()
		if err != nil {
			response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
			return false
		}
		result, err := executor.ExecuteInbound(&plugin.Inbound{
			Request:  r,
			Response: w,
			RawBody:  body,
		})
		if err != nil {
			gw.log.Errorf("failed to execute plugin: %v", err)
			response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
			return false
		}
		if result.Terminated {
			return false
		}
		body = result.Payload
	}

	var event entities.Event
	if err := json.Unmarshal(body, &event); err != nil {
		response.JSON(w, 400, types.ErrorResponse{Message: err.Error()})
		return false
	}

	event.ID = utils.KSUID()
	event.IngestedAt = types.Time{Time: time.Now()}
	event.WorkspaceId = source.WorkspaceId
	if err := event.Validate(); err != nil {
		response.JSON(w, 400, types.ErrorResponse{
			Message: "Request Validation",
			Error:   err,
		})
		return false
	}

	err = gw.ingestEvent(ctx, source.Async, &event)
	if err != nil {
		gw.log.Errorf("failed to ingest event: %v", err)
		response.JSON(w, 500, types.ErrorResponse{Message: "internal error"})
		return false
	}
	if gw.metrics.Enabled {
		gw.metrics.EventTotalCounter.Add(1)
	}

	headers := Headers{}
	headers["Content-Type"] = gw.cfg.Response.ContentType

	if event.Key == nil {
		// returns X-Webhookx-Event-Id header only if key is not present
		headers[constants.HeaderEventId] = event.ID
	}

	if source.Response != nil {
		headers["Content-Type"] = source.Response.ContentType
		exit(w, source.Response.Code, source.Response.Body, headers)
		return true
	}

	// default response
	exit(w, int(gw.cfg.Response.Code), gw.cfg.Response.Body, headers)
	return true
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
			Value:       bytes,
			Time:        time.Now(),
			WorkspaceID: event.WorkspaceId,
		}
		return gw.queue.Enqueue(ctx, &msg)
	}

	return gw.dispatch(ctx, []*entities.Event{event})
}

// Start starts an HTTP server
func (gw *Gateway) Start() {
	gw.ctx, gw.cancel = context.WithCancel(context.Background())

	// warm-up
	gw.dispatcher.WarmUp()

	gw.buildRouter("init")

	if gw.queue != nil {
		gw.queue.StartListen(gw.ctx, gw.HandleMessages)
	}

	schedule.ScheduleWithoutDelay(gw.ctx, func() {
		version := store.GetDefault("router:version", "init").(string)
		if gw.routerVersion == version {
			return
		}
		gw.buildRouter(version)
	}, time.Second)

	if gw.metrics.Enabled && gw.queue != nil {
		schedule.ScheduleWithoutDelay(gw.ctx, func() {
			stats := stats.Stats(gw.queue.Stats())
			size := stats.Int64("eventqueue.size")
			gw.metrics.EventPendingGauge.Set(float64(size))
		}, gw.metrics.Interval)
	}

	gw.bus.Subscribe("source.crud", func(data interface{}) {
		store.Set("router:version", utils.UUID())
	})
	gw.bus.Subscribe("plugin.crud", func(data interface{}) {
		plugin := entities.Plugin{}
		if err := json.Unmarshal(data.(*eventbus.CrudData).Data, &plugin); err != nil {
			zap.S().Errorf("failed to unmarshal event data: %s", err)
			return
		}

		if plugin.SourceId != nil {
			cacheKey := constants.SourcePluginsKey.Build(*plugin.SourceId)
			err := mcache.Invalidate(context.TODO(), cacheKey)
			if err != nil {
				zap.S().Errorf("failed to invalidate cache: key=%s %v", cacheKey, err)
			}
		}
	})

	go func() {
		tls := gw.cfg.TLS
		if tls.Enabled() {
			if err := gw.s.ListenAndServeTLS(tls.Cert, tls.Key); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start gateway HTTPS server: %v", err)
				os.Exit(1)
			}
		} else {
			if err := gw.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start gateway HTTP server: %v", err)
				os.Exit(1)
			}
		}
	}()

	gw.log.Infow(fmt.Sprintf(`listening on address "%s"`, gw.cfg.Listen),
		"tls", gw.cfg.TLS.Enabled(),
	)

}

// Stop stops the HTTP server
func (gw *Gateway) Stop() error {
	gw.cancel()

	if err := gw.s.Shutdown(context.TODO()); err != nil {
		// Error from closing listeners, or context timeout:
		return err
	}

	gw.log.Info("proxy stopped")

	return nil
}

func (gw *Gateway) HandleMessages(ctx context.Context, messages []*queue.Message) error {
	events := make([]*entities.Event, 0, len(messages))
	for _, message := range messages {
		var event entities.Event
		err := json.Unmarshal(message.Value, &event)
		if err != nil {
			gw.log.Warnf("faield to unmarshal message: %v", err)
			continue
		}
		event.WorkspaceId = message.WorkspaceID
		events = append(events, &event)
	}

	err := gw.dispatch(ctx, events)
	if err != nil {
		gw.log.Warnf("failed to dispatch event in batch: %v", err)
	}
	return err
}

func (gw *Gateway) dispatch(ctx context.Context, events []*entities.Event) error {
	attempts, err := gw.dispatcher.Dispatch(ctx, events)
	if err != nil {
		return err
	}
	gw.srv.ScheduleAttempts(ctx, attempts)
	return nil
}

type Headers map[string]string

func exit(w http.ResponseWriter, status int, body string, headers Headers) {
	for _, header := range constants.DefaultResponseHeaders {
		w.Header().Set(header.Name, header.Value)
	}

	if len(headers) > 0 {
		for header, value := range headers {
			w.Header().Set(header, value)
		}
	}

	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func listSourcePlugins(ctx context.Context, db *db.DB, sourceId string) ([]*entities.Plugin, error) {
	// refactor me
	cacheKey := constants.SourcePluginsKey.Build(sourceId)
	plugins, err := mcache.Load(ctx, cacheKey, nil, func(ctx context.Context, id string) (*[]*entities.Plugin, error) {
		plugins, err := db.Plugins.ListSourcePlugin(ctx, id)
		if err != nil {
			return nil, err
		}
		return &plugins, nil
	}, sourceId)
	if err != nil {
		return nil, err
	}
	return *plugins, err
}

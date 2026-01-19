package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/http/middlewares"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/loglimiter"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/queue/redis"
	"github.com/webhookx-io/webhookx/pkg/ratelimiter"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/store"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins"
	"github.com/webhookx-io/webhookx/proxy/router"
	"github.com/webhookx-io/webhookx/services"
	"github.com/webhookx-io/webhookx/services/schedule"
	"github.com/webhookx-io/webhookx/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
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

	cfg *modules.ProxyConfig

	log *zap.SugaredLogger
	s   *http.Server

	router        atomic.Pointer[router.Router]
	routerVersion string

	db *db.DB

	dispatcher *dispatcher.Dispatcher

	queue queue.Queue

	limiter *loglimiter.Limiter

	services *services.Services
}

type Options struct {
	Cfg         *modules.ProxyConfig
	Middlewares []mux.MiddlewareFunc
	DB          *db.DB
	Dispatcher  *dispatcher.Dispatcher
}

func init() {
	stats.Register(stats.ProviderFunc(func() map[string]interface{} {
		return map[string]interface{}{
			"gateway.requests":        counter.Load(),
			"gateway.failed_requests": failures.Load(),
		}
	}))
}

func NewGateway(opts Options, services *services.Services) *Gateway {
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
		cfg:        opts.Cfg,
		log:        zap.S().Named("proxy"),
		db:         opts.DB,
		dispatcher: opts.Dispatcher,
		queue:      q,
		limiter:    loglimiter.NewLimiter(time.Second),
		services:   services,
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

	r.Use(middlewares.NewRecovery(customizeErrorResponse).Handle)
	r.PathPrefix("/").HandlerFunc(gw.Handle)

	gw.s = &http.Server{
		Handler: r,
		Addr:    gw.cfg.Listen,

		ReadTimeout:  time.Duration(gw.cfg.TimeoutRead) * time.Second,
		WriteTimeout: time.Duration(gw.cfg.TimeoutWrite) * time.Second,
	}

	return gw
}

func customizeErrorResponse(err error, w http.ResponseWriter) bool {
	if errors.Is(err, dao.ErrConstraintViolation) {
		response.JSON(w, 400, types.ErrorResponse{Message: err.Error()})
		return true
	}
	return false
}

func (g *Gateway) Name() string {
	return "proxy"
}

func (g *Gateway) buildRouter(version string) {
	ctx, span := tracing.Start(context.Background(), "build_router")
	defer span.End()

	g.log.Debugw("building router", "version", version)

	sources, err := g.db.Sources.List(ctx, &query.SourceQuery{})
	if err != nil {
		g.log.Warnf("failed to build router: %v", err)
		return
	}

	routes := make([]*router.Route, 0)
	for _, source := range sources {
		route := router.Route{
			Paths:   []string{source.Config.HTTP.Path},
			Methods: source.Config.HTTP.Methods,
			Handler: source,
		}
		routes = append(routes, &route)
	}
	g.router.Store(router.NewRouter(routes))
	g.routerVersion = version
}

func (g *Gateway) resolveSource(ctx context.Context, r *http.Request) *entities.Source {
	_, span := tracing.Start(ctx, "resolve_source")
	defer span.End()
	source, _ := g.router.Load().Execute(r).(*entities.Source)
	if source != nil {
		span.SetAttributes(attribute.String("source.id", source.ID))
	}
	return source
}

func (g *Gateway) checkRateLimit(ctx context.Context, source *entities.Source) (ratelimiter.Result, error) {
	d := time.Duration(source.RateLimit.Period) * time.Second
	res, err := g.services.RateLimiter.Allow(ctx, source.ID, source.RateLimit.Quota, d)
	return res, err
}

func (g *Gateway) Handle(w http.ResponseWriter, r *http.Request) {
	res, err := g.handleRequest(w, r)
	if err != nil {
		switch e := err.(type) {
		case *HttpError:
			response.JSON(w, e.Code, types.ErrorResponse{
				Message: e.Message,
				Error:   e.Err,
			})
		case *http.MaxBytesError:
			code := http.StatusRequestEntityTooLarge
			http.Error(w, http.StatusText(code), code)
		default:
			g.log.Errorf("failed to handle request: %v", err)
			response.JSON(w, http.StatusInternalServerError, types.ErrorResponse{Message: "internal error"})
		}

		failures.Add(1)
	}
	if res != nil {
		response.Response(w, res.Headers, res.Code, res.Body)
	}
}

func (g *Gateway) handleRequest(w http.ResponseWriter, r *http.Request) (*Response, error) {
	ctx := context.WithoutCancel(r.Context())

	source := g.resolveSource(ctx, r)
	if source == nil {
		return nil, &HttpError{
			Code:    404,
			Message: "not found",
		}
	}

	ctx = contextx.WithContext(ctx, &contextx.Context{WorkspaceID: source.WorkspaceId})

	if source.RateLimit != nil {
		res, err := g.checkRateLimit(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("failed to rate limiting: %w", err)
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(source.RateLimit.Quota))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(int(math.Ceil(res.Reset.Seconds()))))
		if !res.Allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(res.RetryAfter.Seconds()))))
			return nil, &HttpError{
				Code:    429,
				Message: "rate limit exceeded",
			}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, g.cfg.MaxRequestBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	iterator := plugins.LoadIterator()
	c := plugin.NewContext(ctx, r, w)
	c.SetRequestBody(body)
	for p := range iterator.Iterate(ctx, plugins.PhaseInbound, source.ID) {
		err := p.ExecuteInbound(c)
		if err != nil {
			return nil, fmt.Errorf("failed to execute %s plugin: %v", p.Name(), err)
		}

		if c.IsTerminated() {
			return nil, nil
		}
	}

	var event entities.Event
	if err := json.Unmarshal(c.GetRequestBody(), &event); err != nil {
		return nil, &HttpError{
			Code:    400,
			Message: err.Error(),
		}
	}

	event.ID = utils.KSUID()
	event.IngestedAt = types.Time{Time: time.Now()}
	event.WorkspaceId = source.WorkspaceId
	if err := event.Validate(); err != nil {
		return nil, &HttpError{
			Code:    400,
			Message: "Request Validation",
			Err:     err,
		}
	}

	err = g.ingestEvent(ctx, source.Async, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to ingest event: %w", err)
	}
	if g.services.Metrics.Enabled {
		g.services.Metrics.EventTotalCounter.Add(1)
	}

	res := Response{
		Headers: map[string]string{
			"Content-Type": g.cfg.Response.ContentType,
		},
		Code: int(g.cfg.Response.Code),
		Body: []byte(g.cfg.Response.Body),
	}

	if event.UniqueId == nil {
		// returns X-Webhookx-Event-Id header only if unique_id is not present
		res.Headers[constants.HeaderEventId] = event.ID
	}

	if source.Config.HTTP.Response != nil {
		res.Headers["Content-Type"] = source.Config.HTTP.Response.ContentType
		res.Code = source.Config.HTTP.Response.Code
		res.Body = []byte(source.Config.HTTP.Response.Body)
	}

	return &res, nil
}

func (g *Gateway) ingestEvent(ctx context.Context, async bool, event *entities.Event) error {
	ctx, span := tracing.Start(ctx, "event.ingest")
	span.SetAttributes(attribute.String("event_id", event.ID))
	span.SetAttributes(attribute.Bool("async", async))
	defer span.End()

	if async {
		if g.queue == nil {
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
		return g.queue.Enqueue(ctx, &msg)
	}

	return g.dispatch(ctx, []*entities.Event{event})
}

// Start starts an HTTP server
func (g *Gateway) Start() error {
	g.ctx, g.cancel = context.WithCancel(context.Background())

	// warm-up
	g.dispatcher.WarmUp()

	g.buildRouter("init")

	if g.queue != nil {
		g.queue.StartListen(g.ctx, g.HandleMessages)
	}

	g.services.Scheduler.AddTask(&schedule.Task{
		Name:     "gateway.router_rebuild",
		Interval: time.Second,
		Do: func() {
			version := store.GetDefault("router:version", "init").(string)
			if g.routerVersion == version {
				return
			}
			g.buildRouter(version)
		},
	})

	if g.services.Metrics.Enabled && g.queue != nil {
		g.services.Scheduler.AddTask(&schedule.Task{
			Name:     "gateway.report_metrics",
			Interval: g.services.Metrics.Interval,
			Do: func() {
				stats := stats.Stats(g.queue.Stats())
				size := stats.Int64("eventqueue.size")
				g.services.Metrics.EventPendingGauge.Set(float64(size))
			},
		})
	}

	g.services.EventBus.Subscribe("source.crud", func(data interface{}) {
		store.Set("router:version", utils.UUID())
	})

	go func() {
		tls := g.cfg.TLS
		if tls.Enabled() {
			if err := g.s.ListenAndServeTLS(tls.Cert, tls.Key); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start gateway HTTPS server: %v", err)
				os.Exit(1)
			}
		} else {
			if err := g.s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zap.S().Errorf("Failed to start gateway HTTP server: %v", err)
				os.Exit(1)
			}
		}
	}()

	g.log.Infow(fmt.Sprintf(`listening on address "%s"`, g.cfg.Listen),
		"tls", g.cfg.TLS.Enabled(),
	)

	return nil
}

// Stop stops the HTTP server
func (g *Gateway) Stop(ctx context.Context) error {
	g.log.Infof("exiting")
	g.cancel()

	if err := g.s.Shutdown(ctx); err != nil {
		return err
	}

	g.log.Info("exit")
	return nil
}

func (g *Gateway) HandleMessages(ctx context.Context, messages []*queue.Message) error {
	ctx, span := tracing.Start(ctx, "queue.messages.process")
	defer span.End()

	events := make([]*entities.Event, 0, len(messages))
	for _, message := range messages {
		var event entities.Event
		err := json.Unmarshal(message.Value, &event)
		if err != nil {
			g.log.Warnf("faield to unmarshal message: %v", err)
			continue
		}
		event.WorkspaceId = message.WorkspaceID
		events = append(events, &event)
		sc := trace.SpanContextFromContext(message.GetTraceContext(ctx))
		span.AddLink(trace.Link{SpanContext: sc})
	}

	err := g.dispatch(ctx, events)
	if err != nil {
		g.log.Warnf("failed to dispatch event in batch: %v", err)
	}
	return err
}

func (g *Gateway) dispatch(ctx context.Context, events []*entities.Event) error {
	attempts, err := g.dispatcher.Dispatch(ctx, events)
	if err != nil {
		return err
	}
	g.services.Task.ScheduleAttempts(ctx, attempts)
	return nil
}

package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/admin"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/ratelimiter"
	"github.com/webhookx-io/webhookx/pkg/reports"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/plugins"
	"github.com/webhookx-io/webhookx/proxy"
	"github.com/webhookx-io/webhookx/proxy/middlewares"
	"github.com/webhookx-io/webhookx/service"
	"github.com/webhookx-io/webhookx/status"
	"github.com/webhookx-io/webhookx/status/health"
	"github.com/webhookx-io/webhookx/worker"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	ErrApplicationStarted = errors.New("already started")
	ErrApplicationStopped = errors.New("already stopped")
	Indicators            []*health.Indicator
)

func init() {
	plugins.LoadPlugins()
	entities.LoadOpenAPI(webhookx.OpenAPI)
}

type Application struct {
	nodeID string

	cfg *config.Config

	mux     sync.Mutex
	started bool

	stop chan struct{}

	log     *zap.SugaredLogger
	db      *db.DB
	bus     *eventbus.EventBus
	metrics *metrics.Metrics

	status  *status.Status
	admin   *admin.Admin
	gateway *proxy.Gateway
	worker  *worker.Worker
	tracer  *tracing.Tracer
	srv     *service.Service
}

func New(cfg *config.Config) (*Application, error) {
	app := &Application{
		nodeID: uuid.NewV4().String(),
		cfg:    cfg,
		stop:   make(chan struct{}),
	}

	err := app.initialize()
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *Application) initialize() error {
	cfg := app.cfg
	cfg.OverrideByRole(cfg.Role)

	log, err := log.NewZapLogger(&cfg.Log)
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(log.Desugar())
	app.log = log

	// cache
	client := cfg.Redis.GetClient()
	var c cache.Cache
	if cfg.Role != config.RoleCP {
		c = cache.NewRedisCache(client)
	}
	mcache.Set(mcache.NewMCache(&mcache.Options{
		L1Size: 1000,
		L1TTL:  time.Second * 10,
		L2:     c,
	}))

	sqlDB, err := db.NewSqlDB(cfg.Database)
	if err != nil {
		return err
	}

	app.bus = eventbus.NewEventBus(
		app.NodeID(),
		cfg.Database.GetDSN(),
		log, sqlDB)
	registerEventHandler(app.bus)

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		app.log.Error(err)
	}))

	// tracing
	tracer, err := tracing.New(&cfg.Tracing)
	if err != nil {
		return err
	}

	app.tracer = tracer

	// db
	db, err := db.NewDB(sqlDB, log, app.bus)
	if err != nil {
		return err
	}
	app.db = db
	stats.Register(db)

	app.metrics, err = metrics.New(cfg.Metrics)
	if err != nil {
		return err
	}

	registry := dispatcher.NewRegistry(db)
	app.bus.Subscribe("endpoint.crud", func(v interface{}) {
		data := v.(*eventbus.CrudData)
		registry.Unregister(data.WID)
	})

	dispatcher := dispatcher.NewDispatcher(dispatcher.Options{
		DB:       db,
		Metrics:  app.metrics,
		Registry: registry,
	})

	if cfg.Worker.Enabled || cfg.Proxy.IsEnabled() {
		// queue
		queue := taskqueue.NewRedisQueue(taskqueue.RedisTaskQueueOptions{
			Client: client,
		}, log, app.metrics)
		stats.Register(queue)
		app.srv = service.NewService(service.Options{
			DB:        db,
			TaskQueue: queue,
		})
	}

	// worker
	if cfg.Worker.Enabled {
		d := deliverer.NewHTTPDeliverer(deliverer.Options{
			Logger:         log,
			RequestTimeout: time.Duration(cfg.Worker.Deliverer.Timeout) * time.Millisecond,
			AccessControlOptions: deliverer.AccessControlOptions{
				Deny: cfg.Worker.Deliverer.ACL.Deny,
			},
		})
		if cfg.Worker.Deliverer.Proxy != "" {
			err := d.SetupProxy(deliverer.ProxyOptions{
				URL:              cfg.Worker.Deliverer.Proxy,
				TLSCert:          cfg.Worker.Deliverer.ProxyTLSCert,
				TLSKey:           cfg.Worker.Deliverer.ProxyTLSKey,
				TLSCaCertificate: cfg.Worker.Deliverer.ProxyTLSCaCert,
				TLSVerify:        cfg.Worker.Deliverer.ProxyTLSVerify,
			})
			if err != nil {
				return err
			}
		}
		opts := worker.Options{
			PoolSize:        int(cfg.Worker.Pool.Size),
			PoolConcurrency: int(cfg.Worker.Pool.Concurrency),
			Deliverer:       d,
			DB:              db,
			Srv:             app.srv,
			Tracer:          tracer,
			Metrics:         app.metrics,
			EventBus:        app.bus,
			RedisClient:     client,
		}
		app.worker = worker.NewWorker(opts)
	}

	// admin
	if cfg.Admin.IsEnabled() {
		opts := api.Options{
			Config:     cfg,
			DB:         db,
			Dispatcher: dispatcher,
			EventBus:   app.bus,
		}
		if cfg.AccessLog.Enabled() {
			accessLogger, err := accesslog.NewAccessLogger("admin", accesslog.Options{
				File:   cfg.AccessLog.File,
				Format: string(cfg.AccessLog.Format),
			})
			if err != nil {
				return err
			}
			opts.Middlewares = append(opts.Middlewares, accesslog.NewMiddleware(accessLogger))
		}
		if app.tracer != nil {
			opts.Middlewares = append(opts.Middlewares, otelhttp.NewMiddleware("api.admin"))
		}
		api := api.NewAPI(opts)
		app.admin = admin.NewAdmin(cfg.Admin, api.Handler())
	}

	// gateway
	if cfg.Proxy.IsEnabled() {
		opts := proxy.Options{
			Cfg:         &cfg.Proxy,
			DB:          db,
			Dispatcher:  dispatcher,
			Metrics:     app.metrics,
			Tracer:      tracer,
			EventBus:    app.bus,
			Srv:         app.srv,
			RateLimiter: ratelimiter.NewRedisLimiter(client),
		}
		if cfg.AccessLog.Enabled() {
			accessLogger, err := accesslog.NewAccessLogger("proxy", accesslog.Options{
				File:   cfg.AccessLog.File,
				Format: string(cfg.AccessLog.Format),
			})
			if err != nil {
				return err
			}
			opts.Middlewares = append(opts.Middlewares, accesslog.NewMiddleware(accessLogger))
		}
		if app.tracer != nil {
			opts.Middlewares = append(opts.Middlewares, otelhttp.NewMiddleware("api.proxy"))
		}
		if app.metrics.Enabled {
			opts.Middlewares = append(opts.Middlewares, middlewares.NewMetricsMiddleware(app.metrics).Handle)
		}
		app.gateway = proxy.NewGateway(opts)
	}

	if cfg.Status.IsEnabled() {
		var accessLogger accesslog.AccessLogger
		if cfg.AccessLog.Enabled() {
			accessLogger, err = accesslog.NewAccessLogger("status", accesslog.Options{
				File:   cfg.AccessLog.File,
				Format: string(cfg.AccessLog.Format),
			})
			if err != nil {
				return err
			}
		}
		indicators := make([]*health.Indicator, 0)
		indicators = append(indicators, &health.Indicator{
			Name: "db",
			Check: func() error {
				return db.Ping()
			},
		})
		indicators = append(indicators, &health.Indicator{
			Name: "redis",
			Check: func() error {
				resp := client.Ping(context.TODO())
				if resp.Err() != nil {
					return resp.Err()
				}
				if resp.Val() != "PONG" {
					return errors.New("invalid response from redis: " + resp.Val())
				}
				return nil
			},
		})
		indicators = append(indicators, Indicators...)
		opts := status.Options{
			AccessLog:  accessLogger,
			Config:     cfg,
			Indicators: indicators,
		}
		app.status = status.NewStatus(cfg.Status, app.tracer, opts)
	}

	return nil
}

func registerEventHandler(bus *eventbus.EventBus) {
	bus.ClusteringSubscribe(eventbus.EventCRUD, func(data []byte) {
		eventData := &eventbus.CrudData{}
		if err := json.Unmarshal(data, eventData); err != nil {
			zap.S().Errorf("failed to unmarshal event: %s", err)
			return
		}
		bus.Broadcast(eventbus.EventCRUD, eventData)
	})
	bus.Subscribe(eventbus.EventCRUD, func(data interface{}) {
		eventData := data.(*eventbus.CrudData)
		err := mcache.Invalidate(context.TODO(), eventData.CacheKey)
		if err != nil {
			zap.S().Errorf("failed to invalidate cache: key=%s %v", eventData.CacheKey, err)
		}
		bus.Broadcast(fmt.Sprintf("%s.crud", eventData.Entity), eventData)
	})
}

func (app *Application) DB() *db.DB {
	return app.db
}

func (app *Application) Worker() *worker.Worker {
	return app.worker
}

func (app *Application) NodeID() string {
	return app.nodeID
}

func (app *Application) Config() *config.Config {
	return app.cfg
}

// Start starts application
func (app *Application) Start() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	if app.started {
		return ErrApplicationStarted
	}

	migrator := migrator.New(app.db.DB.DB, nil)
	dbStatus, err := migrator.Status()
	if err != nil {
		return err
	}
	if dbStatus.Dirty {
		return fmt.Errorf("database is in a dirty state at version %d", dbStatus.Version)
	}
	if len(dbStatus.Pendings) > 0 {
		return errors.New("database is not up to date. Run 'webhookx db up' before starting")
	}

	app.log.Infof("starting WebhookX %s", config.VERSION)

	now := time.Now()
	stats.Register(stats.ProviderFunc(func() map[string]interface{} {
		return map[string]interface{}{
			"started_at": now,
		}
	}))

	if err := app.bus.Start(); err != nil {
		return err
	}
	if app.admin != nil {
		app.admin.Start()
	}
	if app.worker != nil {
		app.worker.Start()
	}
	if app.gateway != nil {
		app.gateway.Start()
	}
	if app.status != nil {
		app.status.Start()
	}

	if app.cfg.AnonymousReports {
		reports.Start()
	} else {
		app.log.Info("anonymous reports is disabled")
	}

	app.started = true

	return nil
}

func (app *Application) Wait() {
	<-app.stop
}

// Stop sotps application
func (app *Application) Stop() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	if !app.started {
		return ErrApplicationStopped
	}

	app.log.Info("exiting 👋")

	defer func() {
		app.log.Info("exit")
		_ = app.log.Sync()
	}()

	_ = app.bus.Stop()
	if app.metrics != nil {
		_ = app.metrics.Stop()
	}
	// TODO: timeout
	if app.admin != nil {
		_ = app.admin.Stop()
	}
	if app.worker != nil {
		_ = app.worker.Stop()
	}
	if app.gateway != nil {
		_ = app.gateway.Stop()
	}
	if app.status != nil {
		_ = app.status.Stop()
	}
	if app.tracer != nil {
		_ = app.tracer.Stop()
	}

	app.started = false
	app.stop <- struct{}{}

	return nil
}

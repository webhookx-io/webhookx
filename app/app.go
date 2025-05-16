package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/webhookx-io/webhookx/admin"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/plugins"
	"github.com/webhookx-io/webhookx/proxy"
	"github.com/webhookx-io/webhookx/worker"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	ErrApplicationStarted = errors.New("already started")
	ErrApplicationStopped = errors.New("already stopped")
)

func init() {
	plugins.LoadPlugins()
}

type Application struct {
	mux     sync.Mutex
	started bool

	stop chan struct{}

	cfg *config.Config

	log        *zap.SugaredLogger
	db         *db.DB
	queue      taskqueue.TaskQueue
	dispatcher *dispatcher.Dispatcher
	cache      cache.Cache
	bus        *eventbus.EventBus
	metrics    *metrics.Metrics

	admin   *admin.Admin
	gateway *proxy.Gateway
	worker  *worker.Worker
	tracer  *tracing.Tracer
}

func NewApplication(cfg *config.Config) (*Application, error) {
	app := &Application{
		stop: make(chan struct{}),
		cfg:  cfg,
	}

	err := app.initialize()
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *Application) initialize() error {
	cfg := app.cfg

	log, err := log.NewZapLogger(&cfg.Log)
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(log)
	app.log = zap.S()

	// cache
	client := cfg.Redis.GetClient()
	app.cache = cache.NewRedisCache(client)

	mcache.Set(mcache.NewMCache(&mcache.Options{
		L1Size: 1000,
		L1TTL:  time.Second * 10,
		L2:     app.cache,
	}))

	sqlDB, err := db.NewSqlDB(cfg.Database)
	if err != nil {
		return err
	}

	app.bus = eventbus.NewEventBus(
		app.NodeID(),
		cfg.Database.GetDSN(),
		app.log, sqlDB)
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
	db, err := db.NewDB(sqlDB, app.log, app.bus)
	if err != nil {
		return err
	}
	app.db = db

	app.metrics, err = metrics.New(cfg.Metrics)
	if err != nil {
		return err
	}

	// queue
	queue := taskqueue.NewRedisQueue(taskqueue.RedisTaskQueueOptions{
		Client: client,
	}, app.log, app.metrics)
	app.queue = queue

	app.dispatcher = dispatcher.NewDispatcher(log.Sugar(), queue, db, app.metrics, app.bus)

	// worker
	if cfg.Worker.Enabled {
		opts := worker.WorkerOptions{
			PoolSize:        int(cfg.Worker.Pool.Size),
			PoolConcurrency: int(cfg.Worker.Pool.Concurrency),
		}
		deliverer := deliverer.NewHTTPDeliverer(&cfg.Worker.Deliverer)
		app.worker = worker.NewWorker(opts, db, deliverer, queue, app.metrics, tracer, app.bus)
	}

	// admin
	if cfg.Admin.IsEnabled() {
		var accessLogger accesslog.AccessLogger
		if cfg.AccessLog.Enabled() {
			accessLogger, err = accesslog.NewAccessLogger("admin", accesslog.Options{
				File:   cfg.AccessLog.File,
				Format: string(cfg.AccessLog.Format),
			})
			if err != nil {
				return err
			}
		}
		api := api.NewAPI(cfg, db, app.dispatcher, app.tracer, accessLogger)
		app.admin = admin.NewAdmin(cfg.Admin, api.Handler())
	}

	// gateway
	if cfg.Proxy.IsEnabled() {
		var accessLogger accesslog.AccessLogger
		if cfg.AccessLog.Enabled() {
			accessLogger, err = accesslog.NewAccessLogger("proxy", accesslog.Options{
				File:   cfg.AccessLog.File,
				Format: string(cfg.AccessLog.Format),
			})
			if err != nil {
				return err
			}
		}
		app.gateway = proxy.NewGateway(&cfg.Proxy, db, app.dispatcher, app.metrics, app.tracer, app.bus, accessLogger)
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

func (app *Application) NodeID() string {
	return config.NODE
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

	if err := app.bus.Start(); err != nil {
		return err
	}
	if app.admin != nil {
		app.admin.Start()
	}
	if app.gateway != nil {
		app.gateway.Start()
	}
	if app.worker != nil {
		app.worker.Start()
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

	app.log.Infof("shutting down")

	defer func() {
		app.log.Infof("stopped")
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
	if app.gateway != nil {
		_ = app.gateway.Stop()
	}
	if app.worker != nil {
		_ = app.worker.Stop()
	}
	if app.tracer != nil {
		_ = app.tracer.Stop()
	}

	app.started = false
	app.stop <- struct{}{}

	return nil
}

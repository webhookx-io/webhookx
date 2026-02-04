package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/admin"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/dispatcher"
	"github.com/webhookx-io/webhookx/mcache"
	"github.com/webhookx-io/webhookx/pkg/accesslog"
	"github.com/webhookx-io/webhookx/pkg/cache"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/pkg/metrics"
	"github.com/webhookx-io/webhookx/pkg/ratelimiter"
	"github.com/webhookx-io/webhookx/pkg/reports"
	"github.com/webhookx-io/webhookx/pkg/secret"
	"github.com/webhookx-io/webhookx/pkg/stats"
	"github.com/webhookx-io/webhookx/pkg/store"
	"github.com/webhookx-io/webhookx/pkg/taskqueue"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/plugins"
	"github.com/webhookx-io/webhookx/proxy"
	"github.com/webhookx-io/webhookx/proxy/middlewares"
	"github.com/webhookx-io/webhookx/services"
	"github.com/webhookx-io/webhookx/services/eventbus"
	"github.com/webhookx-io/webhookx/services/schedule"
	"github.com/webhookx-io/webhookx/services/task"
	tracingservice "github.com/webhookx-io/webhookx/services/tracing"
	"github.com/webhookx-io/webhookx/status"
	"github.com/webhookx-io/webhookx/status/health"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

func init() {
	plugins.LoadPlugins()
	entities.LoadOpenAPI(webhookx.OpenAPI)
}

type Application struct {
	id  string
	cfg *config.Config

	mux sync.Mutex

	log *zap.SugaredLogger
	db  *db.DB
	sm  *secret.SecretManager

	services map[string]services.Service
}

func New(cfg *config.Config) (*Application, error) {
	app := &Application{
		id:       uuid.NewV4().String(),
		cfg:      cfg,
		services: make(map[string]services.Service),
	}

	err := app.initialize()
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *Application) initialize() error {
	cfg := app.cfg

	// logger
	if err := app.initLogger(&cfg.Log); err != nil {
		return err
	}

	// cache
	client := cfg.Redis.GetClient()
	if err := app.initCache(cfg, client); err != nil {
		return err
	}

	// sql db
	sqlDB, err := db.NewSqlDB(cfg.Database)
	if err != nil {
		return err
	}

	// event bus
	eventBus := eventbus.NewPostgresEventBus(
		app.id,
		cfg.Database.GetDSN(),
		app.log.Named("eventbus"),
		sqlDB,
	)
	registerEventHandler(eventBus)
	app.registerService(eventBus)

	// scheduler
	scheduler := schedule.NewSchedulerService()
	app.registerScheduledTasks(scheduler)
	app.registerService(scheduler)

	// tracing
	cfg.Tracing.InstanceID = app.id
	if err := tracing.Init(&cfg.Tracing); err != nil {
		return err
	}
	app.registerService(&tracingservice.TracingService{Tracer: tracing.GetTracer()})

	// metrics TODO: refactor this module
	metrics, err := metrics.New(cfg.Metrics, scheduler)
	if err != nil {
		return err
	}
	app.registerService(metrics)

	// db
	db, err := db.NewDB(sqlDB, app.log.Named("db"), eventBus)
	if err != nil {
		return err
	}
	app.db = db
	stats.Register(db)

	dispatcher := dispatcher.NewDispatcher(dispatcher.Options{
		DB:       db,
		Metrics:  metrics,
		Registry: dispatcher.NewRegistry(db),
		EventBus: eventBus,
	})

	// shared services
	services := &services.Services{
		Scheduler:   scheduler,
		EventBus:    eventBus,
		Metrics:     metrics,
		RateLimiter: ratelimiter.NewRedisLimiter(client),
	}

	if cfg.Worker.Enabled || cfg.Proxy.IsEnabled() {
		queue := taskqueue.NewRedisQueue(
			taskqueue.RedisTaskQueueOptions{Client: client},
			app.log,
			metrics,
		)
		stats.Register(queue)
		services.Task = task.NewTaskService(app.log, db, queue)
	}

	// worker
	if err := app.initWorker(&cfg.Worker, services, client); err != nil {
		return err
	}

	// admin
	if err := app.initAdmin(&cfg.Admin, services, dispatcher); err != nil {
		return err
	}

	// gateway
	if err := app.initGateway(&cfg.Proxy, services, dispatcher, metrics); err != nil {
		return err
	}

	// status
	if err := app.initStatus(&cfg.Status, services, client); err != nil {
		return err
	}

	if err := app.initSecretMananger(&cfg.Secret); err != nil {
		return err
	}

	return nil
}

func (app *Application) initLogger(cfg *modules.LogConfig) error {
	logger, err := log.NewZapLogger(cfg)
	if err != nil {
		return err
	}
	app.log = logger
	return nil
}

func (app *Application) initCache(cfg *config.Config, client *redis.Client) error {
	var c cache.Cache
	if cfg.Role != config.RoleCP {
		c = cache.NewRedisCache(client)
	}
	mcache.Set(mcache.NewMCache(&mcache.Options{
		L1Size: 1000,
		L1TTL:  time.Second * 10,
		L2:     c,
	}))
	return nil
}

func (app *Application) initWorker(cfg *modules.WorkerConfig, services *services.Services, client *redis.Client) error {
	if cfg.Enabled {
		delivererOptions := deliverer.Options{
			Logger:         app.log.Named("deliverer"),
			RequestTimeout: time.Duration(cfg.Deliverer.Timeout) * time.Millisecond,
		}
		if cfg.Deliverer.Proxy != "" {
			delivererOptions.ProxyOptions = &deliverer.ProxyOptions{
				URL:              cfg.Deliverer.Proxy,
				TLSCert:          cfg.Deliverer.ProxyTLSCert,
				TLSKey:           cfg.Deliverer.ProxyTLSKey,
				TLSCaCertificate: cfg.Deliverer.ProxyTLSCaCert,
				TLSVerify:        cfg.Deliverer.ProxyTLSVerify,
			}
		}
		if len(cfg.Deliverer.ACL.Deny) > 0 {
			delivererOptions.AclOptions = &deliverer.AclOptions{
				Rules: cfg.Deliverer.ACL.Deny,
			}
		}

		worker := worker.NewWorker(worker.Options{
			PoolSize:         int(cfg.Pool.Size),
			PoolConcurrency:  int(cfg.Pool.Concurrency),
			DelivererOptions: delivererOptions,
			DB:               app.db,
			RedisClient:      client,
		}, services)
		app.registerService(worker)
	}
	return nil
}

func (app *Application) initAdmin(cfg *modules.AdminConfig, services *services.Services, d *dispatcher.Dispatcher) error {
	if cfg.IsEnabled() {
		opts := api.Options{
			Config:     app.cfg,
			DB:         app.db,
			Dispatcher: d,
		}
		if app.cfg.AccessLog.Enabled {
			accessLogger, err := accesslog.NewAccessLogger("admin", accesslog.Options{
				File:    app.cfg.AccessLog.File,
				Format:  string(app.cfg.AccessLog.Format),
				Colored: app.cfg.AccessLog.Colored,
			})
			if err != nil {
				return err
			}
			opts.Middlewares = append(opts.Middlewares, accesslog.NewMiddleware(accessLogger))
		}
		admin := admin.NewAdmin(*cfg, api.NewAPI(opts, services).Handler())
		app.registerService(admin)
	}
	return nil
}

func (app *Application) initGateway(cfg *modules.ProxyConfig, services *services.Services, d *dispatcher.Dispatcher, metrics *metrics.Metrics) error {
	if cfg.IsEnabled() {
		opts := proxy.Options{
			Cfg:        cfg,
			DB:         app.db,
			Dispatcher: d,
		}
		if tracing.Enabled("request") {
			opts.Middlewares = append(opts.Middlewares, otelhttp.NewMiddleware("request"))
		}
		if app.cfg.AccessLog.Enabled {
			accessLogger, err := accesslog.NewAccessLogger("proxy", accesslog.Options{
				File:    app.cfg.AccessLog.File,
				Format:  string(app.cfg.AccessLog.Format),
				Colored: app.cfg.AccessLog.Colored,
			})
			if err != nil {
				return err
			}
			opts.Middlewares = append(opts.Middlewares, accesslog.NewMiddleware(accessLogger))
		}
		if metrics.Enabled {
			opts.Middlewares = append(opts.Middlewares, middlewares.NewMetricsMiddleware(metrics).Handle)
		}
		gateway := proxy.NewGateway(opts, services)
		app.registerService(gateway)
	}
	return nil
}

func (app *Application) initStatus(cfg *modules.StatusConfig, s *services.Services, client *redis.Client) error {
	if cfg.IsEnabled() {
		var accessLogger accesslog.AccessLogger
		var err error
		if app.cfg.AccessLog.Enabled {
			accessLogger, err = accesslog.NewAccessLogger("status", accesslog.Options{
				File:    app.cfg.AccessLog.File,
				Format:  string(app.cfg.AccessLog.Format),
				Colored: app.cfg.AccessLog.Colored,
			})
			if err != nil {
				return err
			}
		}
		indicators := make([]*health.Indicator, 0)
		indicators = append(indicators, &health.Indicator{
			Name: "db",
			Check: func() error {
				return app.db.Ping()
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
		opts := status.Options{
			AccessLog:  accessLogger,
			Indicators: indicators,
		}
		app.registerService(status.NewStatus(*cfg, opts))
	}
	return nil
}

func (app *Application) initSecretMananger(cfg *modules.SecretConfig) error {
	if license.GetLicenser().Allow("secret") && cfg.Enabled() {
		manager, err := secret.NewManagerFromConfig(*cfg)
		if err != nil {
			return err
		}
		manager.WithLogger(app.log.Named("core"))
		app.sm = manager
	}
	return nil
}

func (app *Application) registerService(service services.Service) {
	app.services[service.Name()] = service
}

func (app *Application) getService(name string) services.Service {
	return app.services[name]
}

func (app *Application) buildPluginIterator(version string) (*plugins.Iterator, error) {
	app.log.Debugw("building plugin iterator", "version", version)
	list, err := app.db.Plugins.List(context.TODO(), &query.PluginQuery{})
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins from database: %v", err)
	}
	iterator := plugins.NewIterator(version)
	iterator.WithSecretManager(app.sm)
	if err := iterator.LoadPlugins(list); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %v", err)
	}
	return iterator, nil
}

func (app *Application) scheduleRebuildPluginIterator(bus eventbus.EventBus, scheduler schedule.Scheduler) {
	bus.Subscribe("plugin.crud", func(_ interface{}) {
		store.Set("plugin:version", utils.UUID())
	})

	scheduler.AddTask(&schedule.Task{
		Name:     "app.plugin_rebuild",
		Interval: time.Second,
		Do: func() {
			version := store.GetDefault("plugin:version", "init").(string)
			iterator := plugins.LoadIterator()
			if iterator.Version == version && time.Since(iterator.Created) < time.Minute {
				return
			}

			iterator, err := app.buildPluginIterator(version)
			if err != nil {
				app.log.Error(err)
				return
			}
			plugins.SetIterator(iterator)
		},
	})
}

func registerEventHandler(bus eventbus.EventBus) {
	bus.ClusteringSubscribe(eventbus.EventCRUD, func(data []byte) {
		eventData := &eventbus.CrudData{}
		if err := json.Unmarshal(data, eventData); err != nil {
			zap.S().Errorf("failed to unmarshal event: %s", err)
			return
		}
		bus.Broadcast(context.TODO(), eventbus.EventCRUD, eventData)
	})
	bus.Subscribe(eventbus.EventCRUD, func(d interface{}) {
		data := d.(*eventbus.CrudData)
		cacheKey := constants.CacheKeyFrom(data.CacheName)
		err := mcache.Invalidate(context.TODO(), cacheKey.Build(data.ID))
		if err != nil {
			zap.S().Errorf("failed to invalidate cache: key=%s %v", cacheKey.Build(data.ID), err)
		}
		bus.Broadcast(context.TODO(), fmt.Sprintf("%s.crud", data.Entity), data)
	})
}

func (app *Application) DB() *db.DB {
	return app.db
}

func (app *Application) Config() *config.Config {
	return app.cfg
}

func (app *Application) Scheduler() schedule.Scheduler {
	return app.getService("schedule").(schedule.Scheduler)
}

func (app *Application) registerScheduledTasks(scheduler schedule.Scheduler) {
	if app.cfg.AnonymousReports {
		scheduler.AddTask(&schedule.Task{
			Name:         "anonymous_reports",
			InitialDelay: time.Hour,
			Interval:     time.Hour * 24,
			Do:           reports.Report,
		})
	}

	scheduler.AddTask(&schedule.Task{
		Name:     "license.expiration",
		Interval: time.Hour * 24,
		Do: func() {
			licenser := license.GetLicenser()
			delta := time.Until(licenser.License().ExpiredAt)
			log := app.log.Named("license")
			if delta < 0 {
				log.Errorf("license expired")
			} else if delta < time.Hour*24*30 { // 30 days
				log.Errorf("license will expire at %s", licenser.License().ExpiredAt.Format(time.RFC3339))
			} else if delta < time.Hour*24*90 { // 90 days
				log.Warnf("license will expire on %s", licenser.License().ExpiredAt.Format(time.DateOnly))
			}
		},
	})
}

// Run runs application
func (app *Application) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Start(); err != nil {
		return err
	}

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	return app.stop(ctx)
}

// Start starts application
func (app *Application) Start() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	if app.cfg.Log.Format == modules.LogFormatText && app.cfg.Log.File == "" {
		colored := app.cfg.Log.Colored
		fmt.Println(webhookx.Logo)
		fmt.Println("- Version:", utils.Colorize(webhookx.VERSION, utils.ColorDarkBlue, colored))
		fmt.Println("- Proxy URL:", utils.Colorize(app.cfg.Proxy.URL(), utils.ColorDarkBlue, colored))
		fmt.Println("- Admin URL:", utils.Colorize(app.cfg.Admin.URL(), utils.ColorDarkBlue, colored))
		fmt.Println("- Status URL:", utils.Colorize(app.cfg.Status.URL(), utils.ColorDarkBlue, colored))
		fmt.Println("- Worker:", utils.Colorize(app.cfg.Worker.Status(), utils.ColorDarkBlue, colored))
		fmt.Println()
	}

	migrate := migrator.New(app.db.DB.DB, nil)
	dbStatus, err := migrate.Status()
	if err != nil {
		return fmt.Errorf("failed to check db status: %w", err)
	}
	if dbStatus.Dirty {
		return fmt.Errorf("database is in a dirty state at version %d", dbStatus.Version)
	}
	if len(dbStatus.Pendings) > 0 {
		return errors.New("database is not up to date. Run 'webhookx db up' before starting")
	}

	app.log.Infof("starting WebhookX %s", webhookx.VERSION)

	now := time.Now()
	stats.Register(stats.ProviderFunc(func() map[string]interface{} {
		return map[string]interface{}{
			"started_at": now,
		}
	}))

	if app.getService("worker") != nil || app.getService("proxy") != nil {
		iterator, err := app.buildPluginIterator("init")
		if err != nil {
			return fmt.Errorf("failed to build plugin iterator: %s", err)
		}
		plugins.SetIterator(iterator)
		app.scheduleRebuildPluginIterator(
			app.getService("eventbus").(eventbus.EventBus),
			app.getService("schedule").(schedule.Scheduler),
		)
	}

	if !app.cfg.AnonymousReports {
		app.log.Info("anonymous reports is disabled")
	}

	services := []services.Service{
		app.getService("eventbus"),
		app.getService("metrics"),
		app.getService("admin"),
		app.getService("worker"),
		app.getService("proxy"),
		app.getService("status"),
		app.getService("schedule"),
	}
	return startServices(services...)
}

// Stop sotps application
func (app *Application) Stop() error {
	app.mux.Lock()
	defer app.mux.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	err := app.stop(ctx)
	return err
}

func (app *Application) stop(ctx context.Context) error {
	app.log.Info("exiting 👋")

	defer func() { _ = app.log.Sync() }()

	var errs []error
	errs = append(errs, stopServices(ctx, time.Second*10, app.getService("proxy"), app.getService("admin"), app.getService("status")))
	errs = append(errs, stopServices(ctx, 0, app.getService("eventbus")))
	errs = append(errs, stopServices(ctx, 0, app.getService("worker")))
	errs = append(errs, stopServices(ctx, time.Second*5, app.getService("metrics"), app.getService("tracing")))
	errs = append(errs, stopServices(ctx, 0, app.getService("schedule")))
	errs = append(errs, app.db.Close())

	app.log.Info("exit")

	return errors.Join(errs...)
}

func startServices(services ...services.Service) error {
	for _, s := range services {
		if s != nil {
			if err := s.Start(); err != nil {
				return fmt.Errorf("failed to start service '%s': %w", s.Name(), err)
			}
		}
	}
	return nil
}

func stopServices(ctx context.Context, timeout time.Duration, services ...services.Service) error {
	var mu sync.Mutex
	var errs []error
	var g sync.WaitGroup

	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	for _, s := range services {
		if s != nil {
			g.Go(func() {
				if err := s.Stop(ctx); err != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("failed to stop service '%s': %w", s.Name(), err))
					mu.Unlock()
				}
			})
		}
	}
	g.Wait()

	return errors.Join(errs...)
}

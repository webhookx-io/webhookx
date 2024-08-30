package app

import (
	"context"
	"github.com/webhookx-io/webhookx/admin"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/proxy"
	"github.com/webhookx-io/webhookx/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os/signal"
	"syscall"
	"time"
)

type Application struct {
	ctx      context.Context
	cancel   func()
	stopChan chan bool

	cfg *config.Config

	log   *zap.SugaredLogger
	db    *db.DB
	queue queue.TaskQueue

	admin   *admin.Admin
	gateway *proxy.Gateway
	worker  *worker.Worker
}

func NewApplication(cfg *config.Config) (*Application, error) {
	ctx, cancel := context.WithCancel(context.Background())

	app := &Application{
		ctx:      ctx,
		cancel:   cancel,
		cfg:      cfg,
		stopChan: make(chan bool, 1),
	}

	err := app.initialize()
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (app *Application) initialize() error {
	cfg := app.cfg

	// log
	level, err := zapcore.ParseLevel(cfg.Log.Level)
	if err != nil {
		return err
	}
	log, err := zap.NewDevelopment(
		zap.AddStacktrace(zap.PanicLevel),
		zap.IncreaseLevel(level))
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(log)
	app.log = zap.S()

	// db
	db, err := db.NewDB(&cfg.DatabaseConfig)
	if err != nil {
		return err
	}
	app.db = db

	// queue
	client := cfg.RedisConfig.GetClient()
	queue := queue.NewRedisQueue(client)
	app.queue = queue

	// worker
	if cfg.WorkerConfig.Enabled {
		app.worker = worker.NewWorker(app.ctx, &cfg.WorkerConfig, db, queue)
	}

	// server
	if cfg.AdminConfig.IsEnabled() {
		handler := api.NewAPI(cfg, db, queue).Handler()
		app.admin = admin.NewAdmin(cfg.AdminConfig, handler)
	}

	// gateway
	if cfg.ProxyConfig.IsEnabled() {
		app.gateway = proxy.NewGateway(&cfg.ProxyConfig, db, queue)
	}

	return nil
}

func (app *Application) Start() error {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		app.log.Infof("shutting down")
		app.Stop()
	}()

	if app.admin != nil {
		app.admin.Start()
	}
	if app.gateway != nil {
		app.gateway.Start()
	}
	if app.worker != nil {
		app.worker.Start()
	}

	app.wait()

	time.Sleep(time.Second)
	return nil
}

func (app *Application) wait() {
	<-app.stopChan
}

func (app *Application) Stop() {
	defer func() {
		app.log.Infof("stopped")
	}()

	app.cancel()

	if app.admin != nil {
		app.admin.Stop()
	}
	if app.gateway != nil {
		app.gateway.Stop()
	}
	if app.worker != nil {
		app.worker.Stop()
	}

	app.stopChan <- true
}

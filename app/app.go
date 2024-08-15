package app

import (
	"github.com/webhookx-io/webhookx/api"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/server"
	"github.com/webhookx-io/webhookx/worker"
)

type App struct {
	cfg *config.Config

	db    *db.DB
	queue queue.TaskQueue

	server *server.Server
	worker *worker.Worker
}

func NewApp(cfg *config.Config) (*App, error) {
	app := &App{
		cfg: cfg,
	}
	err := app.init()
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (app *App) init() error {
	cfg := app.cfg
	// db
	db, err := db.NewDB(cfg)
	if err != nil {
		return err
	}
	app.db = db

	// queue
	client := cfg.RedisConfig.GetClient()
	queue := queue.NewRedisQueue(client)
	app.queue = queue

	// worker
	app.worker = worker.NewWorker(cfg, db, queue)

	// server
	handler := api.NewAPI(cfg, db, queue).Handler()
	app.server = server.NewServer(cfg.AdminConfig, handler)
	return nil
}

func (app *App) Start() error {
	app.worker.Start()

	app.server.Start()
	defer app.server.Close()

	app.server.Wait()
	return nil
}

func (app *App) Stop() error {
	return nil
}

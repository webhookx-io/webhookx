package helper

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"time"

	"github.com/creasty/defaults"
	"github.com/go-resty/resty/v2"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/utils"
)

var cfg *config.Config

func init() {
	var err error
	cfg, err = config.Init()
	if err != nil {
		panic(err)
	}
}

var defaultEnvs = map[string]string{
	"WEBHOOKX_LOG_LEVEL":  "debug",
	"WEBHOOKX_LOG_FORMAT": "text",
	"WEBHOOKX_LOG_FILE":   "webhookx.log",
}

// Start starts WebhookX with given environment variables
func Start(envs map[string]string) (*app.Application, error) {
	for name, value := range defaultEnvs {
		if _, ok := envs[name]; !ok {
			err := os.Setenv(name, value)
			if err != nil {
				return nil, err
			}
		}
	}
	for name, value := range envs {
		err := os.Setenv(name, value)
		if err != nil {
			return nil, err
		}
	}

	cfg, err := config.Init()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(defaultEnvs["WEBHOOKX_LOG_FILE"]); err == nil {
		TruncateFile(defaultEnvs["WEBHOOKX_LOG_FILE"])
	}

	app, err := app.NewApplication(cfg)
	if err != nil {
		return nil, err
	}
	if err := app.Start(); err != nil {
		return nil, err
	}

	go app.Wait()

	time.Sleep(time.Second)
	return app, nil
}

func AdminClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8080")
	return c
}

func ProxyClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8081")
	return c
}

func DB() *db.DB {
	cfg, err := config.Init()
	if err != nil {
		return nil
	}
	db, err := db.NewDB(&cfg.DatabaseConfig)
	if err != nil {
		return nil
	}
	return db
}

type EntitiesConfig struct {
	Endpoints      []*entities.Endpoint
	Sources        []*entities.Source
	Events         []*entities.Event
	Attempts       []*entities.Attempt
	AttemptDetails []*entities.AttemptDetail
	Plugins        []*entities.Plugin
}

func InitDB(truncated bool, entities *EntitiesConfig) *db.DB {
	if truncated {
		err := ResetDB()
		if err != nil {
			panic(err)
		}
	}

	db, err := db.NewDB(&cfg.DatabaseConfig)
	if err != nil {
		panic(err)
	}

	if entities == nil {
		return db
	}

	ws, err := db.Workspaces.GetDefault(context.TODO())
	if err != nil {
		panic(err)
	}

	for _, e := range entities.Endpoints {
		e.WorkspaceId = ws.ID
		err = db.Endpoints.Insert(context.TODO(), e)
		if err != nil {
			panic(err)
		}
	}

	for _, e := range entities.Sources {
		e.WorkspaceId = ws.ID
		err = db.Sources.Insert(context.TODO(), e)
		if err != nil {
			panic(err)
		}
	}

	for _, e := range entities.Events {
		e.WorkspaceId = ws.ID
		err = db.Events.Insert(context.TODO(), e)
		if err != nil {
			panic(err)
		}
	}

	for _, e := range entities.Attempts {
		e.WorkspaceId = ws.ID
		err = db.Attempts.Insert(context.TODO(), e)
		if err != nil {
			panic(err)
		}
	}

	for _, e := range entities.AttemptDetails {
		e.WorkspaceId = ws.ID
		err = db.AttemptDetails.Insert(context.TODO(), e)
		if err != nil {
			panic(err)
		}
	}

	for _, e := range entities.Plugins {
		e.WorkspaceId = ws.ID
		err = db.Plugins.Insert(context.TODO(), e)
		if err != nil {
			panic(err)
		}
	}

	return db
}

func ResetDB() error {
	cfg, err := config.Init()
	if err != nil {
		return err
	}

	migrator := migrator.New(&cfg.DatabaseConfig)
	err = migrator.Reset()
	if err != nil {
		return err
	}
	return migrator.Up()
}

func TruncateFile(filename string) {
	err := os.Truncate(filename, 0)
	if err != nil {
		panic("failed to truncate file: " + err.Error())
	}
}

func FileLine(filename string, n int) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for i := 1; scanner.Scan(); i++ {
		s := scanner.Text()
		if i == n {
			return s, nil
		}
	}

	return "", nil
}

func FileHasLine(filename string, regex string) (bool, error) {
	file, err := os.Open(filename)
	if err != nil {
		return false, err
	}
	defer file.Close()

	r, err := regexp.Compile(regex)
	if err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if r.MatchString(line) {
			return true, nil
		}
	}

	return false, nil
}

func DefaultEndpoint() *entities.Endpoint {
	var entity entities.Endpoint
	entity.Init()
	defaults.Set(&entity)

	entity.Request = entities.RequestConfig{
		URL:    "http://localhost:9999/anything",
		Method: "POST",
	}
	entity.Retry.Config.Attempts = []int64{0, 3, 3}
	entity.Events = []string{"foo.bar"}

	return &entity
}

func DefaultSource() *entities.Source {
	var entity entities.Source
	entity.Init()
	defaults.Set(&entity)

	entity.Path = "/"
	entity.Methods = []string{"POST"}

	return &entity
}

func DefaultEvent() *entities.Event {
	var entity entities.Event
	defaults.Set(&entity)

	entity.ID = utils.KSUID()
	entity.EventType = "foo.bar"
	entity.Data = []byte("{}")

	return &entity
}

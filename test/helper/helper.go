package helper

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"github.com/go-resty/resty/v2"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/test"
	"os"
	"regexp"
	"time"
)

var (
	OtelCollectorTracesFile  = test.FilePath("output/otel/traces.json")
	OtelCollectorMetricsFile = test.FilePath("output/otel/metrics.json")
)

var defaultEnvs = map[string]string{
	"WEBHOOKX_LOG_LEVEL":       "debug",
	"WEBHOOKX_LOG_FORMAT":      "text",
	"WEBHOOKX_LOG_FILE":        "webhookx.log",
	"WEBHOOKX_ACCESS_LOG_FILE": "webhookx.log",
}

func setEnvs(envs map[string]string) error {
	for name, value := range envs {
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return nil
}

// Start starts WebhookX with given environment variables
func Start(envs map[string]string) (*app.Application, error) {
	if err := setEnvs(defaultEnvs); err != nil {
		return nil, err
	}

	if err := setEnvs(envs); err != nil {
		return nil, err
	}

	cfg, err := config.Init()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(defaultEnvs["WEBHOOKX_LOG_FILE"]); err == nil {
		TruncateFile(defaultEnvs["WEBHOOKX_LOG_FILE"])
	}

	if _, err := os.Stat(defaultEnvs["WEBHOOKX_ACCESS_LOG_FILE"]); err == nil {
		TruncateFile(defaultEnvs["WEBHOOKX_ACCESS_LOG_FILE"])
	}

	app, err := app.New(cfg)
	if err != nil {
		return nil, err
	}
	if err := app.Start(); err != nil {
		return nil, err
	}

	go func() {
		app.Wait()
		for name := range envs {
			os.Unsetenv(name)
		}
	}()

	time.Sleep(time.Second)
	return app, nil
}

func AdminClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8080")
	return c
}

func AdminTLSClient() *resty.Client {
	c := resty.New()
	c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	c.SetBaseURL("https://localhost:8080")
	return c
}

func ProxyClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8081")
	return c
}

func ProxyTLSClient() *resty.Client {
	c := resty.New()
	c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	c.SetBaseURL("https://localhost:8081")
	return c
}

func StatusClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL("http://localhost:8082")
	return c
}

func DB() *db.DB {
	cfg, err := config.Init()
	if err != nil {
		return nil
	}
	sqlDB, err := db.NewSqlDB(cfg.Database)
	if err != nil {
		return nil
	}
	logger, err := log.NewZapLogger(&cfg.Log)
	if err != nil {
		return nil
	}
	eventbus := eventbus.NewEventBus(
		uuid.NewV4().String(),
		cfg.Database.GetDSN(),
		logger, sqlDB)

	db, err := db.NewDB(sqlDB, logger, eventbus)
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
		err := resetDB()
		if err != nil {
			panic(err)
		}
	}

	db := DB()

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

func GetDeafultWorkspace() (*entities.Workspace, error) {
	db := DB()
	return db.Workspaces.GetDefault(context.TODO())
}

func resetDB() error {
	cfg, err := config.Init()
	if err != nil {
		return err
	}

	sqlDB, err := db.NewSqlDB(cfg.Database)
	if err != nil {
		return err
	}

	migrator := migrator.New(sqlDB, &migrator.Options{Quiet: true})
	err = migrator.Reset()
	if err != nil {
		return err
	}
	return migrator.Up()
}

func TruncateFile(filename string) error {
	return os.Truncate(filename, 0)
}

func FileLine(filename string, n int) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for i := 1; scanner.Scan(); i++ {
		s := scanner.Text()
		if i == n {
			return s, nil
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return "", nil
}

func FileCountLine(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	n := 0
	for scanner.Scan() {
		scanner.Text()
		n++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return n, nil
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

	const maxCapacity = 2 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	for scanner.Scan() {
		line := scanner.Text()
		if r.MatchString(line) {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	return false, nil
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
}

func InitOtelOutput() {
	if !PathExist(OtelCollectorTracesFile) {
		os.Create(OtelCollectorTracesFile)
	}
	if !PathExist(OtelCollectorMetricsFile) {
		os.Create(OtelCollectorMetricsFile)
	}
}

func GenerateTraceID() string {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(traceID)
}

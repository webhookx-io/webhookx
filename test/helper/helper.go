package helper

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"database/sql"
	"encoding/hex"
	"fmt"
	"maps"
	"net"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-resty/resty/v2"
	vault "github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/cmd"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/migrator"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/pkg/log"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/utils"
)

var (
	ProxyHttpURL  = "http://localhost:9700"
	ProxyHttpsURL = "https://localhost:9700"
	AdminHttpURL  = "http://localhost:9701"
	AdminHttpsURL = "https://localhost:9701"
	StatusHttpURL = "http://localhost:9702"

	LogFile                  = test.FilePath("webhookx.log")
	OtelCollectorTracesFile  = test.FilePath("output/otel/traces.json")
	OtelCollectorMetricsFile = test.FilePath("output/otel/metrics.json")

	// Environments is default test environments
	Environments = map[string]string{
		"NO_COLOR":                           "true",
		"WEBHOOKX_LOG_LEVEL":                 "debug",
		"WEBHOOKX_LOG_FORMAT":                "text",
		"WEBHOOKX_LOG_FILE":                  LogFile,
		"WEBHOOKX_LOG_COLORED":               "false",
		"WEBHOOKX_ACCESS_LOG_FILE":           LogFile,
		"WEBHOOKX_ACCESS_LOG_COLORED":        "false",
		"WEBHOOKX_WORKER_DELIVERER_ACL_DENY": "",
		"WEBHOOKX_PROXY_LISTEN":              "127.0.0.1:9700",
		"WEBHOOKX_ADMIN_LISTEN":              "127.0.0.1:9701",
		"WEBHOOKX_STATUS_LISTEN":             "127.0.0.1:9702",
		"WEBHOOKX_DATABASE_DATABASE":         "webhookx_test",
		"WEBHOOKX_WORKER_POOL_SIZE":          "100",
		"WEBHOOKX_WORKER_POOL_CONCURRENCY":   "10",
	}
)

// NewTestEnv returns a map that with default test environment variables set.
func NewTestEnv(sets map[string]string) map[string]string {
	env := make(map[string]string)
	maps.Copy(env, Environments)
	maps.Copy(env, sets)
	return env
}

// SetEnvs sets env variables and returns a function to restore environment variables
func SetEnvs(sets map[string]string) func() {
	env := make(map[string]string)
	maps.Copy(env, sets)
	originals := make(map[string]*string)
	for k, v := range env {
		old, existed := os.LookupEnv(k)
		if existed {
			originals[k] = &old
		} else {
			originals[k] = nil
		}
		_ = os.Setenv(k, v)
	}
	return func() {
		for k, old := range originals {
			if old == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *old)
			}
		}
	}
}

type Options struct {
	Licenser license.Licenser
}

type Option func(*Options)

func WithLicenser(licenser license.Licenser) Option {
	return func(opts *Options) {
		opts.Licenser = licenser
	}
}

func MustStart(envs map[string]string, opts ...Option) *app.Application {
	app, err := Start(envs, opts...)
	if err != nil {
		panic(err)
	}
	return app
}

// Start starts application with given environment variables
func Start(envs map[string]string, opts ...Option) (application *app.Application, err error) {
	envs = NewTestEnv(envs)
	cancel := SetEnvs(envs)

	defer func() {
		if err != nil {
			cancel()
		}
	}()

	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	if options.Licenser == nil {
		lic, err := license.Load()
		if err != nil {
			return nil, err
		}
		options.Licenser = license.NewLicenser(lic)
	}

	license.SetLicenser(options.Licenser)

	cfg := config.New()
	if err := config.Load("", cfg); err != nil {
		return nil, errors.Wrap(err, "could not load configuration")
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	if _, err := os.Stat(cfg.Log.File); err == nil {
		TruncateFile(cfg.Log.File)
	}

	if _, err := os.Stat(cfg.AccessLog.File); err == nil {
		TruncateFile(cfg.Log.File)
	}

	application, err = app.New(cfg)
	if err != nil {
		return
	}
	if err := application.Start(); err != nil {
		return nil, err
	}

	go func() {
		application.Wait()
		cancel()
	}()

	time.Sleep(time.Second)
	return application, nil
}

// ExecAppCommand executes application command
func ExecAppCommand(args ...string) (output string, err error) {
	cancel := SetEnvs(Environments)
	defer cancel()

	root := cmd.NewRootCmd()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return buf.String(), err
}

func AdminClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL(AdminHttpURL)
	c.DisableWarn = true
	return c
}

func AdminTLSClient() *resty.Client {
	c := resty.New()
	c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	c.SetBaseURL(AdminHttpsURL)
	return c
}

func ProxyClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL(ProxyHttpURL)
	c.DisableWarn = true
	return c
}

func ProxyTLSClient() *resty.Client {
	c := resty.New()
	c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	c.SetBaseURL(ProxyHttpsURL)
	return c
}

func StatusClient() *resty.Client {
	c := resty.New()
	c.SetBaseURL(StatusHttpURL)
	c.DisableWarn = true
	return c
}

func NewDB(cfg *config.Config) *db.DB {
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

type TestEntities struct {
	Endpoints      []*entities.Endpoint
	Sources        []*entities.Source
	Events         []*entities.Event
	Attempts       []*entities.Attempt
	AttemptDetails []*entities.AttemptDetail
	Plugins        []*entities.Plugin
}

func (t *TestEntities) AddEndpoint(endpoint *entities.Endpoint) {
	t.Endpoints = append(t.Endpoints, endpoint)
}
func (t *TestEntities) AddSource(source *entities.Source) {
	t.Sources = append(t.Sources, source)
}
func (t *TestEntities) AddEvent(event *entities.Event) {
	t.Events = append(t.Events, event)
}
func (t *TestEntities) AddAttempt(attempt *entities.Attempt) {
	t.Attempts = append(t.Attempts, attempt)
}
func (t *TestEntities) AddAttemptDetail(attemptDetail *entities.AttemptDetail) {
	t.AttemptDetails = append(t.AttemptDetails, attemptDetail)
}
func (t *TestEntities) AddPlugin(plugin *entities.Plugin) {
	t.Plugins = append(t.Plugins, plugin)
}

func InitDB(truncated bool, entities *TestEntities) *db.DB {
	cfg, err := LoadConfig(LoadConfigOptions{
		Envs: NewTestEnv(nil),
	})
	if err != nil {
		panic(err)
	}

	db := NewDB(cfg)

	if truncated {
		err := resetDB(db.SqlDB())
		if err != nil {
			panic(err)
		}
		err = resetRedis(cfg.Redis.GetClient())
		if err != nil {
			panic(err)
		}
		err = resetRedis(cfg.Proxy.Queue.Redis.GetClient())
		if err != nil {
			panic(err)
		}
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
		for _, p := range e.Plugins {
			p.EndpointId = utils.Pointer(e.ID)
			p.WorkspaceId = ws.ID
			err = db.Plugins.Insert(context.TODO(), p)
			if err != nil {
				panic(err)
			}
		}
	}

	for _, s := range entities.Sources {
		s.WorkspaceId = ws.ID
		err = db.Sources.Insert(context.TODO(), s)
		if err != nil {
			panic(err)
		}
		for _, p := range s.Plugins {
			p.SourceId = utils.Pointer(s.ID)
			p.WorkspaceId = ws.ID
			err = db.Plugins.Insert(context.TODO(), p)
			if err != nil {
				panic(err)
			}
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
	cfg, err := LoadConfig(LoadConfigOptions{
		Envs: NewTestEnv(nil),
	})
	if err != nil {
		return nil, err
	}
	db := NewDB(cfg)
	return db.Workspaces.GetDefault(context.TODO())
}

func resetDB(db *sql.DB) error {
	m := migrator.New(db, &migrator.Options{Quiet: true})
	err := m.Reset()
	if err != nil {
		return err
	}
	return m.Up()
}

func resetRedis(redis *redis.Client) error {
	cmd := redis.FlushDB(context.TODO())
	return cmd.Err()
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

func WaitForServer(urlstring string, timeout time.Duration) error {
	u, err := url.Parse(urlstring)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", u.Host, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("server at %s not ready after %v", u.Host, timeout)
}

func VaultClient() *vault.Client {
	cfg := vault.DefaultConfig()
	cfg.Address = "http://127.0.0.1:8200"
	client, err := vault.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	client.SetToken("root")
	return client
}

func SecretManangerClient() *secretsmanager.Client {
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithBaseEndpoint("http://localhost:4566"),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			"test",
			"test",
			"",
		))),
	)
	if err != nil {
		panic(err)
	}
	client := secretsmanager.NewFromConfig(cfg, func(options *secretsmanager.Options) {})
	return client
}

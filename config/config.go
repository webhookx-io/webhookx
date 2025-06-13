package config

import (
	"encoding/json"
	"github.com/creasty/defaults"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx/pkg/envconfig"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

var (
	VERSION = "dev"
	COMMIT  = "unknown"
	NODE    = uuid.NewV4().String()
)

type Config struct {
	Log       LogConfig       `yaml:"log" json:"log" envconfig:"LOG"`
	AccessLog AccessLogConfig `yaml:"access_log" json:"access_log" envconfig:"ACCESS_LOG"`
	Database  DatabaseConfig  `yaml:"database" json:"database" envconfig:"DATABASE"`
	Redis     RedisConfig     `yaml:"redis" json:"redis" envconfig:"REDIS"`
	Admin     AdminConfig     `yaml:"admin" json:"admin" envconfig:"ADMIN"`
	Proxy     ProxyConfig     `yaml:"proxy" json:"proxy" envconfig:"PROXY"`
	Worker    WorkerConfig    `yaml:"worker" json:"worker" envconfig:"WORKER"`
	Metrics   MetricsConfig   `yaml:"metrics" json:"metrics" envconfig:"METRICS"`
	Tracing   TracingConfig   `yaml:"tracing" json:"tracing" envconfig:"TRACING"`
}

func (cfg Config) String() string {
	bytes, err := json.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

func (cfg Config) Validate() error {
	if err := cfg.Log.Validate(); err != nil {
		return err
	}
	if err := cfg.AccessLog.Validate(); err != nil {
		return err
	}
	if err := cfg.Database.Validate(); err != nil {
		return err
	}
	if err := cfg.Redis.Validate(); err != nil {
		return err
	}
	if err := cfg.Admin.Validate(); err != nil {
		return err
	}
	if err := cfg.Proxy.Validate(); err != nil {
		return err
	}
	if err := cfg.Worker.Validate(); err != nil {
		return err
	}
	if err := cfg.Metrics.Validate(); err != nil {
		return err
	}
	if err := cfg.Tracing.Validate(); err != nil {
		return err
	}

	return nil
}

func Init() (*Config, error) {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return nil, err
	}

	err := envconfig.Process("WEBHOOKX", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func InitWithFile(filename string) (*Config, error) {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		return nil, err
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil && err != io.EOF {
		return nil, err
	}

	err = envconfig.Process("WEBHOOKX", &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

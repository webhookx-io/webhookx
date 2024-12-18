package config

import (
	"encoding/json"
	"github.com/creasty/defaults"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx/pkg/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

var (
	VERSION = "dev"
	COMMIT  = "unknown"
	NODE    = uuid.NewV4().String()
)

var cfg Config

type Config struct {
	Log      LogConfig      `yaml:"log" envconfig:"LOG"`
	Database DatabaseConfig `yaml:"database" envconfig:"DATABASE"`
	Redis    RedisConfig    `yaml:"redis" envconfig:"REDIS"`
	Admin    AdminConfig    `yaml:"admin" envconfig:"ADMIN"`
	Proxy    ProxyConfig    `yaml:"proxy" envconfig:"PROXY"`
	Worker   WorkerConfig   `yaml:"worker" envconfig:"WORKER"`
	Metrics  MetricsConfig  `yaml:"metrics" envconfig:"METRICS"`
	Tracing  TracingConfig  `yaml:"tracing" envconfig:"TRACING"`
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
	if err := defaults.Set(&cfg); err != nil {
		return nil, err
	}

	err := envconfig.Process("WEBHOOKX", &cfg)
	if err != nil {
		return nil, err
	}
	cfg.injectTracingEnabled()
	return &cfg, nil
}

func InitWithFile(filename string) (*Config, error) {
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
	if err != nil {
		return nil, err
	}

	err = envconfig.Process("WEBHOOKX", &cfg)
	if err != nil {
		return nil, err
	}
	cfg.injectTracingEnabled()
	return &cfg, nil
}

func (cfg *Config) injectTracingEnabled() {
	cfg.Database.SetTracingEnabled(cfg.Tracing.Enabled)
}

package config

import (
	"encoding/json"
	"fmt"
	"github.com/creasty/defaults"
	uuid "github.com/satori/go.uuid"
	"github.com/webhookx-io/webhookx/pkg/envconfig"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"slices"
)

var (
	VERSION = "dev"
	COMMIT  = "unknown"
	NODE    = uuid.NewV4().String()
)

type Role string

const (
	RoleAll      Role = "all"
	RoleCP       Role = "cp"
	RoleDPWorker Role = "dp_worker"
	RoleDPProxy  Role = "dp_proxy"
)

type Config struct {
	Log       LogConfig       `yaml:"log" json:"log" envconfig:"LOG"`
	AccessLog AccessLogConfig `yaml:"access_log" json:"access_log" envconfig:"ACCESS_LOG"`
	Database  DatabaseConfig  `yaml:"database" json:"database" envconfig:"DATABASE"`
	Redis     RedisConfig     `yaml:"redis" json:"redis" envconfig:"REDIS"`
	Admin     AdminConfig     `yaml:"admin" json:"admin" envconfig:"ADMIN"`
	Status    StatusConfig    `yaml:"status" json:"status" envconfig:"STATUS"`
	Proxy     ProxyConfig     `yaml:"proxy" json:"proxy" envconfig:"PROXY"`
	Worker    WorkerConfig    `yaml:"worker" json:"worker" envconfig:"WORKER"`
	Metrics   MetricsConfig   `yaml:"metrics" json:"metrics" envconfig:"METRICS"`
	Tracing   TracingConfig   `yaml:"tracing" json:"tracing" envconfig:"TRACING"`
	Role      Role            `yaml:"role" json:"role" envconfig:"ROLE" default:"all"`
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
	if err := cfg.Status.Validate(); err != nil {
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

	if !slices.Contains([]Role{RoleAll, RoleCP, RoleDPWorker, RoleDPProxy}, cfg.Role) {
		return fmt.Errorf("invalid role: %s", cfg.Role)
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

func (cfg *Config) OverrideByRole(role Role) {
	switch role {
	case RoleCP:
		if cfg.Admin.Listen == "" {
			cfg.Admin.Listen = "127.0.0.1:8080"
		}
		cfg.Proxy.Listen = ""
		cfg.Worker.Enabled = false
	case RoleDPProxy:
		if cfg.Proxy.Listen == "" {
			cfg.Proxy.Listen = "127.0.0.1:8081"
		}
		cfg.Admin.Listen = ""
		cfg.Worker.Enabled = false
	case RoleDPWorker:
		cfg.Admin.Listen = ""
		cfg.Proxy.Listen = ""
		cfg.Worker.Enabled = true
	}
}

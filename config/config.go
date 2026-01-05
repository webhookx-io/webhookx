package config

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/config/types"
)

type Role string

const (
	RoleStandalone Role = "standalone"
	RoleCP         Role = "cp"
	RoleDPWorker   Role = "dp_worker"
	RoleDPProxy    Role = "dp_proxy"
)

var _ types.Config = &Config{}

// Config Configuration
type Config struct {
	modules.BaseConfig
	Log              modules.LogConfig       `yaml:"log" json:"log" envconfig:"LOG"`
	AccessLog        modules.AccessLogConfig `yaml:"access_log" json:"access_log" envconfig:"ACCESS_LOG"`
	Database         modules.DatabaseConfig  `yaml:"database" json:"database" envconfig:"DATABASE"`
	Redis            modules.RedisConfig     `yaml:"redis" json:"redis" envconfig:"REDIS"`
	Admin            modules.AdminConfig     `yaml:"admin" json:"admin" envconfig:"ADMIN"`
	Status           modules.StatusConfig    `yaml:"status" json:"status" envconfig:"STATUS"`
	Proxy            modules.ProxyConfig     `yaml:"proxy" json:"proxy" envconfig:"PROXY"`
	Worker           modules.WorkerConfig    `yaml:"worker" json:"worker" envconfig:"WORKER"`
	Metrics          modules.MetricsConfig   `yaml:"metrics" json:"metrics" envconfig:"METRICS"`
	Tracing          modules.TracingConfig   `yaml:"tracing" json:"tracing" envconfig:"TRACING"`
	Role             Role                    `yaml:"role" json:"role" envconfig:"ROLE" default:"standalone"`
	AnonymousReports bool                    `yaml:"anonymous_reports" json:"anonymous_reports" envconfig:"ANONYMOUS_REPORTS" default:"true"`
	Secret           modules.SecretConfig    `yaml:"secret" json:"secret" envconfig:"SECRET"`
}

func (cfg *Config) PostProcess() error {
	switch cfg.Role {
	case RoleCP:
		if cfg.Admin.Listen == "" {
			cfg.Admin.Listen = "127.0.0.1:9601"
		}
		cfg.Proxy.Listen = ""
		cfg.Worker.Enabled = false
	case RoleDPProxy:
		if cfg.Proxy.Listen == "" {
			cfg.Proxy.Listen = "0.0.0.0:9600"
		}
		cfg.Admin.Listen = ""
		cfg.Worker.Enabled = false
	case RoleDPWorker:
		cfg.Admin.Listen = ""
		cfg.Proxy.Listen = ""
		cfg.Worker.Enabled = true
	}
	return nil
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
	if !slices.Contains([]Role{RoleStandalone, RoleCP, RoleDPWorker, RoleDPProxy}, cfg.Role) {
		return fmt.Errorf("invalid role: '%s'", cfg.Role)
	}
	if err := cfg.Secret.Validate(); err != nil {
		return err
	}

	return nil
}

func New() *Config {
	var cfg Config
	if err := defaults.Set(&cfg); err != nil {
		panic(err)
	}
	return &cfg
}

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/pkg/envconfig"
	"github.com/webhookx-io/webhookx/pkg/secret"
	"github.com/webhookx-io/webhookx/pkg/secret/reference"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

var (
	VERSION = "dev"
	COMMIT  = "unknown"
)

type Role string

const (
	RoleStandalone Role = "standalone"
	RoleCP         Role = "cp"
	RoleDPWorker   Role = "dp_worker"
	RoleDPProxy    Role = "dp_proxy"
)

type Config struct {
	Log              LogConfig       `yaml:"log" json:"log" envconfig:"LOG"`
	AccessLog        AccessLogConfig `yaml:"access_log" json:"access_log" envconfig:"ACCESS_LOG"`
	Database         DatabaseConfig  `yaml:"database" json:"database" envconfig:"DATABASE"`
	Redis            RedisConfig     `yaml:"redis" json:"redis" envconfig:"REDIS"`
	Admin            AdminConfig     `yaml:"admin" json:"admin" envconfig:"ADMIN"`
	Status           StatusConfig    `yaml:"status" json:"status" envconfig:"STATUS"`
	Proxy            ProxyConfig     `yaml:"proxy" json:"proxy" envconfig:"PROXY"`
	Worker           WorkerConfig    `yaml:"worker" json:"worker" envconfig:"WORKER"`
	Metrics          MetricsConfig   `yaml:"metrics" json:"metrics" envconfig:"METRICS"`
	Tracing          TracingConfig   `yaml:"tracing" json:"tracing" envconfig:"TRACING"`
	Role             Role            `yaml:"role" json:"role" envconfig:"ROLE" default:"standalone"`
	AnonymousReports bool            `yaml:"anonymous_reports" json:"anonymous_reports" envconfig:"ANONYMOUS_REPORTS" default:"true"`
	Secret           SecretConfig    `yaml:"secret" json:"secret" envconfig:"SECRET"`
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

type Options struct {
	YAML []byte
}

func resolveReferences(n *yaml.Node, manager *secret.Manager) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if reference.IsReference(n.Value) {
			ref, err := reference.Parse(n.Value)
			if err != nil {
				return err
			}
			val, err := manager.ResolveReference(context.TODO(), ref)
			if err != nil {
				return err
			}
			n.Value = val
		}
	case yaml.MappingNode:
		for i := 0; i < len(n.Content); i += 2 {
			if err := resolveReferences(n.Content[i+1], manager); err != nil {
				return err
			}
		}
	case yaml.AliasNode:
		if n.Alias != nil {
			if err := resolveReferences(n.Alias, manager); err != nil {
				return err
			}
		}
	default:
		for _, c := range n.Content {
			if err := resolveReferences(c, manager); err != nil {
				return err
			}
		}
	}
	return nil
}

func New(opts *Options) (*Config, error) {
	var cfg Config
	err := defaults.Set(&cfg)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &Options{}
	}

	var doc *yaml.Node
	if len(opts.YAML) > 0 {
		doc = new(yaml.Node)
		if err := yaml.Unmarshal(opts.YAML, doc); err != nil {
			return nil, err
		}
	}

	// todo: only if secret feature is allowed
	if doc != nil {
		// fixme
		if len(doc.Content) > 0 {
			if node := utils.FindYaml(doc.Content[0], "secret"); node != nil {
				if err := node.Decode(&cfg.Secret); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := envconfig.Process("WEBHOOKX_SECRET", &cfg.Secret); err != nil {
		return nil, err
	}

	var reader = envconfig.EnvironmentReader

	if err := cfg.Secret.Validate(); err != nil {
		return nil, err
	}

	if cfg.Secret.Enabled() {
		providers := make(map[string]map[string]interface{})
		for _, p := range cfg.Secret.Providers {
			name := string(p)
			providers[name] = cfg.Secret.GetProviderConfiguration(name)
		}
		manager := secret.NewManager(zap.S(), providers)
		reader = func(key string) (string, bool, error) {
			value, ok, _ := envconfig.EnvironmentReader.Read(key)
			if ok && reference.IsReference(value) {
				ref, err := reference.Parse(value)
				if err != nil {
					return "", false, err
				}
				resolved, err := manager.ResolveReference(context.TODO(), ref)
				if err != nil {
					return "", false, err
				}
				value = resolved
			}
			return value, ok, nil
		}

		if doc != nil {
			if len(doc.Content) > 0 {
				if err := resolveReferences(doc.Content[0], manager); err != nil {
					return nil, err
				}
			}
			resolvedYaml := utils.Must(yaml.Marshal(doc))
			fmt.Println(string(resolvedYaml))
			doc = new(yaml.Node)
			_ = yaml.Unmarshal(resolvedYaml, doc)
		}
	}

	if doc != nil {
		if err := doc.Decode(&cfg); err != nil {
			return nil, err
		}
	}

	err = envconfig.ProcessWithReader("WEBHOOKX", &cfg, reader)
	return &cfg, err
}

func (cfg *Config) OverrideByRole(role Role) {
	switch role {
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
}

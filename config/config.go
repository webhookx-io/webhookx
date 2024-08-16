package config

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/pkg/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

var (
	VERSION = "dev"
	COMMIT  = "unknown"
)

var cfg Config

type Config struct {
	Log            LogConfig      `yaml:"log" envconfig:"LOG"`
	DatabaseConfig DatabaseConfig `yaml:"database" envconfig:"DATABASE"`
	RedisConfig    RedisConfig    `yaml:"redis" envconfig:"REDIS"`
	AdminConfig    AdminConfig    `yaml:"admin" envconfig:"ADMIN"`
	ProxyConfig    ProxyConfig    `yaml:"proxy" envconfig:"PROXY"`
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
	if err := cfg.DatabaseConfig.Validate(); err != nil {
		return err
	}
	if err := cfg.RedisConfig.Validate(); err != nil {
		return err
	}
	if err := cfg.AdminConfig.Validate(); err != nil {
		return err
	}
	if err := cfg.ProxyConfig.Validate(); err != nil {
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

	return &cfg, nil
}

package config

import (
	"fmt"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host     string   `yaml:"host" json:"host" default:"localhost"`
	Port     uint32   `yaml:"port" json:"port" default:"6379"`
	Password Password `yaml:"password" json:"password" default:""`
	Database uint32   `yaml:"database" json:"database" default:"0"`
	// fixme: pool property
}

func (cfg RedisConfig) GetClient() *redis.Client {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: string(cfg.Password),
		DB:       int(cfg.Database),
	}
	return redis.NewClient(options)
}

func (cfg RedisConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}

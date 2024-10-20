package config

import (
	"fmt"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisConfig struct {
	Host     string `yaml:"host" default:"localhost"`
	Port     uint32 `yaml:"port" default:"6379"`
	Password string `yaml:"password" default:""`
	Database uint32 `yaml:"database" default:"0"`
	// fixme: pool property
}

func (cfg RedisConfig) GetClient() *redis.Client {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       int(cfg.Database),
	}
	rdb := redis.NewClient(options)
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		zap.S().Errorf(`failed to instrument redis otel tracing %v`, err)
	}
	return rdb
}

func (cfg RedisConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}

package config

import (
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

type RedisConfig struct {
	Host        string   `yaml:"host" json:"host" default:"localhost"`
	Port        uint32   `yaml:"port" json:"port" default:"6379"`
	Password    Password `yaml:"password" json:"password" default:""`
	Database    uint32   `yaml:"database" json:"database" default:"0"`
	MaxPoolSize uint32   `yaml:"max_pool_size" json:"max_pool_size" default:"0"`
}

func (cfg RedisConfig) GetClient() *redis.Client {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: string(cfg.Password),
		DB:       int(cfg.Database),
		PoolSize: int(cfg.MaxPoolSize),
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	}
	return redis.NewClient(options)
}

func (cfg RedisConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}

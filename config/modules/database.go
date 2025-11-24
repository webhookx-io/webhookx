package modules

import (
	"fmt"

	"github.com/webhookx-io/webhookx/config/types"
)

type DatabaseConfig struct {
	BaseConfig
	Host        string         `yaml:"host" json:"host" default:"localhost"`
	Port        uint32         `yaml:"port" json:"port" default:"5432"`
	Username    string         `yaml:"username" json:"username" default:"webhookx"`
	Password    types.Password `yaml:"password" json:"password" default:""`
	Database    string         `yaml:"database" json:"database" default:"webhookx"`
	Parameters  string         `yaml:"parameters" json:"parameters" default:"application_name=webhookx&sslmode=disable&connect_timeout=10"`
	MaxPoolSize uint32         `yaml:"max_pool_size" json:"max_pool_size" default:"40" envconfig:"MAX_POOL_SIZE"`
	MaxLifetime uint32         `yaml:"max_life_time" json:"max_life_time" default:"1800" envconfig:"MAX_LIFETIME"`
}

func (cfg DatabaseConfig) GetDSN() string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
	if len(cfg.Parameters) > 0 {
		dsn = fmt.Sprintf("%s?%s", dsn, cfg.Parameters)
	}
	return dsn
}

func (cfg DatabaseConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}

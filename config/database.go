package config

import (
	"fmt"
)

type DatabaseConfig struct {
	Host     string `yaml:"host" default:"localhost"`
	Port     uint32 `yaml:"port" default:"5432"`
	Username string `yaml:"username" default:"webhookx"`
	Password string `yaml:"password" default:""`
	Database string `yaml:"database" default:"webhookx"`
}

func (cfg DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
}

func (cfg DatabaseConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}

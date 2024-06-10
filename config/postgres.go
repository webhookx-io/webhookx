package config

import (
	"database/sql"
	"fmt"
)

type PostgresConfig struct {
	Host     string `default:"localhost"`
	Port     uint32 `default:"5432"`
	Username string `default:"webhookx"`
	Password string `default:""`
	Database string `default:"webhookx"`
}

func (cfg PostgresConfig) GetDB() (*sql.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
	)
	return sql.Open("postgres", dsn)
}

func (cfg PostgresConfig) Validate() error {
	if cfg.Port > 65535 {
		return fmt.Errorf("port must be in the range [0, 65535]")
	}
	return nil
}

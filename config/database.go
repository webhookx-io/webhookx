package config

import (
	"database/sql"
	"fmt"
	"github.com/XSAM/otelsql"
	"time"
)

type DatabaseConfig struct {
	Host           string `yaml:"host" default:"localhost"`
	Port           uint32 `yaml:"port" default:"5432"`
	Username       string `yaml:"username" default:"webhookx"`
	Password       string `yaml:"password" default:""`
	Database       string `yaml:"database" default:"webhookx"`
	Parameters     string `yaml:"parameters" default:"application_name=webhookx&sslmode=disable&connect_timeout=10"`
	MaxPoolSize    uint32 `yaml:"max_pool_size" default:"40" envconfig:"MAX_POOL_SIZE"`
	MaxLifetime    uint32 `yaml:"max_life_time" default:"1800" envconfig:"MAX_LIFETIME"`
	tracingEnabled bool
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

func (cfg *DatabaseConfig) SetTracingEnabled(enabled bool) {
	cfg.tracingEnabled = enabled
}

func (cfg *DatabaseConfig) InitSqlDB() (*sql.DB, error) {
	var driverName = "postgres"
	var err error
	if cfg.tracingEnabled {
		driverName, err = otelsql.Register(driverName,
			otelsql.WithSpanOptions(otelsql.SpanOptions{
				OmitConnResetSession: true,
				OmitConnPrepare:      true,
				OmitConnectorConnect: true,
				OmitRows:             true,
			}))
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open(driverName, cfg.GetDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(int(cfg.MaxPoolSize))
	db.SetMaxIdleConns(int(cfg.MaxPoolSize))
	db.SetConnMaxLifetime(time.Second * time.Duration(cfg.MaxLifetime))
	return db, nil
}

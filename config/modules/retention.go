package modules

import (
	"fmt"
	"time"

	"github.com/webhookx-io/webhookx/config/types"
)

type RetentionConfig struct {
	BaseConfig
	Enabled  bool               `yaml:"enabled" json:"enabled" default:"false"`
	Interval types.Duration     `yaml:"interval" json:"interval" default:"24h"`
	TTL      RetentionTTLConfig `yaml:"ttl" json:"ttl"`
}

type RetentionTTLConfig struct {
	Events   types.Duration `yaml:"events" json:"events" default:"0"`
	Attempts types.Duration `yaml:"attempts" json:"attempts" default:"0"`
}

func (cfg RetentionConfig) Validate() error {
	if cfg.Interval.Duration() < time.Hour {
		return fmt.Errorf("minimum interval is 1h")
	}
	if cfg.TTL.Events < 0 {
		return fmt.Errorf("ttl.events cannot be negative")
	}
	if cfg.TTL.Events > 0 && cfg.TTL.Events.Duration() < time.Hour*24 {
		return fmt.Errorf("minimum ttl.events is 1d")
	}
	if cfg.TTL.Attempts < 0 {
		return fmt.Errorf("ttl.attempts cannot be negative")
	}
	if cfg.TTL.Attempts > 0 && cfg.TTL.Attempts.Duration() < time.Hour*24 {
		return fmt.Errorf("minimum ttl.attempts is 1d")
	}
	return nil
}

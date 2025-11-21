package modules

import (
	"fmt"
	"slices"

	"github.com/webhookx-io/webhookx/config/core"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJson LogFormat = "json"
)

type LogConfig struct {
	core.BaseConfig
	Level   LogLevel  `yaml:"level" json:"level" default:"info"`
	Format  LogFormat `yaml:"format" json:"format" default:"text"`
	Colored bool      `yaml:"colored" json:"colored" default:"true"`
	File    string    `yaml:"file" json:"file"`
}

func (cfg LogConfig) Validate() error {
	if !slices.Contains([]LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}, cfg.Level) {
		return fmt.Errorf("invalid level: %s", cfg.Level)
	}
	if !slices.Contains([]LogFormat{LogFormatText, LogFormatJson}, cfg.Format) {
		return fmt.Errorf("invalid format: %s", cfg.Format)
	}
	return nil
}

package config

import (
	"fmt"
	"slices"
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
	File   string    `yaml:"file" default:"/dev/stdout"`
	Level  LogLevel  `yaml:"level" default:"info"`
	Format LogFormat `yaml:"format" default:"text"`
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

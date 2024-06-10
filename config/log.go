package config

import (
	"fmt"
	"slices"
)

type LogLevel = string

const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

type LogFormat = string

const (
	LogFormatText = "text"
	LogFormatJson = "json"
)

type LogConfig struct {
	Level  LogLevel `default:"INFO"`
	Format string   `default:"text"`
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

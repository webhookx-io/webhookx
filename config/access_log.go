package config

import (
	"fmt"
	"slices"
)

type AccessLogConfig struct {
	File   string    `yaml:"file" default:"/dev/stdout"`
	Format LogFormat `yaml:"format" default:"text"`
}

func (cfg AccessLogConfig) Validate() error {
	if !slices.Contains([]LogFormat{LogFormatText, LogFormatJson}, cfg.Format) {
		return fmt.Errorf("invalid format: %s", cfg.Format)
	}
	return nil
}

func (cfg AccessLogConfig) Enabled() bool {
	return cfg.File != ""
}

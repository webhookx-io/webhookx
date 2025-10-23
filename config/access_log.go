package config

import (
	"fmt"
	"slices"
)

type AccessLogConfig struct {
	Enabled bool      `yaml:"enabled" json:"enabled" default:"true"`
	Format  LogFormat `yaml:"format" json:"format" default:"text"`
	Colored bool      `yaml:"colored" json:"colored" default:"true"`
	File    string    `yaml:"file" json:"file"`
}

func (cfg AccessLogConfig) Validate() error {
	if !slices.Contains([]LogFormat{LogFormatText, LogFormatJson}, cfg.Format) {
		return fmt.Errorf("invalid format: %s", cfg.Format)
	}
	return nil
}

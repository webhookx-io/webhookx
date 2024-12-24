package config

import (
	"fmt"
	"slices"
)

type MetricsConfig struct {
	Attributes    Map           `yaml:"attributes" envconfig:"ATTRIBUTES"`
	Exports       []Export      `yaml:"exports" envconfig:"EXPORTS"`
	PushInterval  uint32        `yaml:"push_interval" default:"10"`
	Opentelemetry Opentelemetry `yaml:"opentelemetry"`
}

func (cfg *MetricsConfig) Validate() error {
	if err := cfg.Opentelemetry.Validate(); err != nil {
		return err
	}
	for _, export := range cfg.Exports {
		if !slices.Contains([]Export{ExportOpenTelemetry}, export) {
			return fmt.Errorf("invalid export: %s", export)
		}
	}
	if cfg.Opentelemetry.PushInterval != 0 {
		cfg.PushInterval = cfg.Opentelemetry.PushInterval
	}
	if cfg.PushInterval < 1 || cfg.PushInterval > 60 {
		return fmt.Errorf("interval must be in the range [1, 60]")
	}
	return nil
}

type Export string

const (
	ExportOpenTelemetry Export = "opentelemetry"
)

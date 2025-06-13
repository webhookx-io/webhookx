package config

import (
	"errors"
)

type TracingConfig struct {
	Enabled       bool          `yaml:"enabled" json:"enabled" default:"false"`
	Attributes    Map           `yaml:"attributes" json:"attributes"`
	Opentelemetry Opentelemetry `yaml:"opentelemetry" json:"opentelemetry"`
	SamplingRate  float64       `yaml:"sampling_rate" json:"sampling_rate" default:"1.0" envconfig:"SAMPLING_RATE"`
}

func (cfg TracingConfig) Validate() error {
	if cfg.SamplingRate > 1 || cfg.SamplingRate < 0 {
		return errors.New("sampling_rate must be in the range [0, 1]")
	}
	if err := cfg.Opentelemetry.Validate(); err != nil {
		return err
	}
	return nil
}

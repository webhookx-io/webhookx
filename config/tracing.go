package config

import (
	"errors"
	"fmt"
	"slices"
)

type TracingConfig struct {
	Enabled       bool                 `yaml:"enabled" json:"enabled" default:"false"`
	Attributes    Map                  `yaml:"attributes" json:"attributes"`
	Opentelemetry OpentelemetryTracing `yaml:"opentelemetry" json:"opentelemetry"`
	SamplingRate  float64              `yaml:"sampling_rate" json:"sampling_rate" default:"1.0" envconfig:"SAMPLING_RATE"`
}

type OpentelemetryTracing struct {
	Protocol OtlpProtocol `yaml:"protocol" json:"protocol" envconfig:"PROTOCOL" default:"http/protobuf"`
	Endpoint string       `yaml:"endpoint" json:"endpoint" envconfig:"ENDPOINT" default:"http://localhost:4318/v1/traces"`
}

func (cfg OpentelemetryTracing) Validate() error {
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}
	return nil
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

package config

import (
	"fmt"
	"slices"
)

type MetricsConfig struct {
	Attributes    Map           `yaml:"attributes" envconfig:"ATTRIBUTES"`
	Exports       []Export      `yaml:"exports" envconfig:"EXPORTS"`
	OpenTelemetry Opentelemetry `yaml:"opentelemetry" envconfig:"OPENTELEMETRY"`
}

func (cfg *MetricsConfig) Validate() error {
	if err := cfg.OpenTelemetry.Validate(); err != nil {
		return err
	}
	for _, export := range cfg.Exports {
		if !slices.Contains([]Export{ExportOpenTelemetry}, export) {
			return fmt.Errorf("invalid export: %s", export)
		}
	}
	return nil
}

type Export string

const (
	ExportOpenTelemetry Export = "opentelemetry"
)

type OtlpProtocol string

const (
	OtlpProtocolGRPC OtlpProtocol = "grpc"
	OtlpProtocolHTTP OtlpProtocol = "http/protobuf"
)

type Opentelemetry struct {
	PushInterval uint32       `yaml:"push_interval" default:"10"`
	Protocol     OtlpProtocol `yaml:"protocol" envconfig:"PROTOCOL" default:"http/protobuf"`
	Endpoint     string       `yaml:"endpoint" envconfig:"ENDPOINT" default:"http://localhost:4318/v1/metrics"`
}

func (cfg Opentelemetry) Validate() error {
	if cfg.PushInterval < 1 || cfg.PushInterval > 60 {
		return fmt.Errorf("interval must be in the range [1, 60]")
	}
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}
	return nil
}

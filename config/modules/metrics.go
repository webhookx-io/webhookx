package modules

import (
	"fmt"
	"slices"

	"github.com/webhookx-io/webhookx/config/types"
)

type MetricsConfig struct {
	BaseConfig
	Attributes    types.Map            `yaml:"attributes" json:"attributes"`
	Exports       []Export             `yaml:"exports" json:"exports"`
	PushInterval  uint32               `yaml:"push_interval" json:"push_interval" default:"10" envconfig:"PUSH_INTERVAL"`
	Opentelemetry OpentelemetryMetrics `yaml:"opentelemetry" json:"opentelemetry"`
}

type OpentelemetryMetrics struct {
	Protocol OtlpProtocol `yaml:"protocol" json:"protocol" envconfig:"PROTOCOL" default:"http/protobuf"`
	Endpoint string       `yaml:"endpoint" json:"endpoint" envconfig:"ENDPOINT" default:"http://localhost:4318/v1/metrics"`
}

func (cfg OpentelemetryMetrics) Validate() error {
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}
	return nil
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
	if cfg.PushInterval < 1 || cfg.PushInterval > 60 {
		return fmt.Errorf("interval must be in the range [1, 60]")
	}
	return nil
}

type Export string

const (
	ExportOpenTelemetry Export = "opentelemetry"
)

package modules

import (
	"errors"
	"fmt"
	"slices"

	"github.com/webhookx-io/webhookx/config/types"
)

type TracingConfig struct {
	BaseConfig
	InstanceID       string               `yaml:"-" json:"-"`
	Instrumentations []string             `yaml:"instrumentations" json:"instrumentations"`
	Attributes       types.Map            `yaml:"attributes" json:"attributes"`
	Opentelemetry    OpentelemetryTracing `yaml:"opentelemetry" json:"opentelemetry"`
	SamplingRate     float64              `yaml:"sampling_rate" json:"sampling_rate" default:"1.0" envconfig:"SAMPLING_RATE"`
}

func (cfg *TracingConfig) Enabled() bool {
	return len(cfg.Instrumentations) > 0
}

func (cfg *TracingConfig) Validate() error {
	if cfg.SamplingRate > 1 || cfg.SamplingRate < 0 {
		return errors.New("sampling_rate must be in the range [0, 1]")
	}
	if err := cfg.Opentelemetry.Validate(); err != nil {
		return err
	}
	for _, str := range cfg.Instrumentations {
		if !slices.Contains([]string{"request", "plugin", "dao", "@all"}, str) {
			return fmt.Errorf("invalid instrumentations: %s", str)
		}

	}
	return nil
}

type OpentelemetryTracing struct {
	Protocol OtlpProtocol `yaml:"protocol" json:"protocol" envconfig:"PROTOCOL" default:"http/protobuf"`
	Endpoint string       `yaml:"endpoint" json:"endpoint" envconfig:"ENDPOINT" default:"http://127.0.0.1:4318/v1/traces"`
}

func (cfg OpentelemetryTracing) Validate() error {
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}
	return nil
}

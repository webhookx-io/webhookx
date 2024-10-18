package config

import (
	"fmt"
	"slices"
)

type MetricsConfig struct {
	Attributes    Map           `yaml:"attributes" envconfig:"ATTRIBUTES"`
	Exporters     []string      `yaml:"exporters" envconfig:"EXPORTERS"`
	OpenTelemetry Opentelemetry `yaml:"opentelemetry" envconfig:"OPENTELEMETRY"`
}

func (cfg *MetricsConfig) Validate() error {
	if err := cfg.OpenTelemetry.Validate(); err != nil {
		return err
	}
	return nil
}

//type Datadog struct {
//	Address  string `yaml:"address" default:"udp://127.0.0.1:8125"`
//	Prefix   string `yaml:"prefix" default:"webhookx"`
//	Interval uint32 `yaml:"interval" default:"10"`
//}
//
//func (cfg Datadog) Validate() error {
//	if cfg.Address != "" {
//		if !(strings.HasPrefix(cfg.Address, "udp://") || strings.HasPrefix(cfg.Address, "unix://")) {
//			return fmt.Errorf("address must start with udp:// or unix://")
//		}
//	}
//	if cfg.Interval > 60 {
//		return fmt.Errorf("interval must be in the range [0, 60]")
//	}
//	return nil
//}

//func (cfg Datadog) Decode(value string) error {
//	if value == "" {
//		return nil
//	}
//	return nil
//}

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
	if cfg.PushInterval > 60 {
		return fmt.Errorf("interval must be in the range [0, 60]")
	}
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}
	return nil
}

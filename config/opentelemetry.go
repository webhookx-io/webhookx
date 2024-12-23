package config

import (
	"fmt"
	"slices"
)

type OtlpProtocol string

const (
	OtlpProtocolGRPC OtlpProtocol = "grpc"
	OtlpProtocolHTTP OtlpProtocol = "http/protobuf"
)

type Opentelemetry struct {
	Protocol OtlpProtocol `yaml:"protocol" envconfig:"PROTOCOL" default:"http/protobuf"`
	Endpoint string       `yaml:"endpoint" envconfig:"ENDPOINT" default:"http://localhost:4318/v1/metrics"`
}

func (cfg Opentelemetry) Validate() error {
	if !slices.Contains([]OtlpProtocol{OtlpProtocolGRPC, OtlpProtocolHTTP}, cfg.Protocol) {
		return fmt.Errorf("invalid protocol: %s", cfg.Protocol)
	}
	return nil
}

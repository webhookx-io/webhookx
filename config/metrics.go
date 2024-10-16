package config

import (
	"fmt"
	"strings"
)

type MetricsConfig struct {
	Datadog *Datadog       `yaml:"datadog"`
	OTLP    *Opentelemetry `yaml:"otlp"`
}

func (cfg MetricsConfig) Validate() error {
	if cfg.Datadog != nil {
		if err := cfg.Datadog.Validate(); err != nil {
			return err
		}
	}
	if cfg.OTLP != nil {
		if err := cfg.OTLP.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Datadog struct {
	Address  string `yaml:"address" default:"udp://127.0.0.1:8125"`
	Prefix   string `yaml:"prefix" default:"webhookx"`
	Interval uint32 `yaml:"interval" default:"10"`
}

func (cfg Datadog) Validate() error {
	if cfg.Address != "" {
		if !(strings.HasPrefix(cfg.Address, "udp://") || strings.HasPrefix(cfg.Address, "unix://")) {
			return fmt.Errorf("address must start with udp:// or unix://")
		}
	}
	if cfg.Interval > 60 {
		return fmt.Errorf("interval must be in the range [0, 60]")
	}
	return nil
}

type Opentelemetry struct {
	Interval uint32    `yaml:"interval" default:"10"`
	GRPC     *OTLPgRPC `yaml:"grpc"`
	HTTP     *OTLPHttp `yaml:"http"`

	// FIXME
	ExplicitBoundaries []float64 `yaml:"explicit_boundaries"`
}

func (cfg Opentelemetry) Validate() error {
	if cfg.Interval > 60 {
		return fmt.Errorf("interval must be in the range [0, 60]")
	}
	return nil
}

type OTLPHttp struct {
	Endpoint string            `yaml:"endpoint" default:"http://localhost:4318/v1/traces"`
	Headers  map[string]string `yaml:"headers"`
}

type OTLPgRPC struct {
	Endpoint string            `yaml:"endpoint" default:"http://localhost:4317"`
	Headers  map[string]string `yaml:"headers"`
	Insecure bool              `yaml:"insecure" default:"false"`
}

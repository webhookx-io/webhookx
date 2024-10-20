package config

import (
	"errors"
)

type TracingConfig struct {
	GlobalAttributes        map[string]string    `yaml:"global_attributes"`
	Opentelemetry           *OpenTelemetryConfig `yaml:"opentelemetry"`
	ServiceName             string               `yaml:"service_name" default:"WebhookX"`
	CapturedRequestHeaders  []string             `yaml:"captured_request_headers"`
	CapturedResponseHeaders []string             `yaml:"captured_response_headers"`
	SafeQueryParams         []string             `yaml:"safe_query_params"`
	SamplingRate            float64              `yaml:"sampling_rate" default:"1"`
}

type OpenTelemetryConfig struct {
	HTTP OtelEndpoint `yaml:"http,omitempty"`
	GRPC OtelEndpoint `yaml:"grpc,omitempty"`
}

type OtelEndpoint struct {
	Headers  map[string]string `yaml:"headers,omitempty"`
	Endpoint string            `yaml:"endpoint"`
}

func (cfg TracingConfig) Validate() error {
	if cfg.SamplingRate > 1 || cfg.SamplingRate < 0 {
		return errors.New("invalid sampling rate, must be [0,1]")
	}
	return nil
}

func (cfg TracingConfig) IsEnable() bool {
	otelConfig := cfg.Opentelemetry
	if otelConfig == nil {
		return false
	}

	if otelConfig.HTTP.Endpoint != "" || otelConfig.GRPC.Endpoint != "" {
		return true
	}

	return false

}

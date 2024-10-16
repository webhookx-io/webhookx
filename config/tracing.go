package config

import (
	"errors"

	"github.com/webhookx-io/webhookx/pkg/types"
)

type TracingConfig struct {
	GlobalAttributes        map[string]string    `yaml:"global_attributes"`
	Opentelemetry           *OpenTelemetryConfig `yaml:"opentelemetry"`
	ServiceName             string               `yaml:"service_name" default:"WebhookX"`
	CapturedRequestHeaders  []string             `yaml:"captured_request_headers"`
	CapturedResponseHeaders []string             `yaml:"captured_response_headers"`
	SafeQueryParams         []string             `yaml:"safe_query_params"`
	SamplingRate            float64              `yaml:"sampling_rate" default:"1"`
	AddInternals            bool                 `yaml:"add_internals" default:"false"`
}

type OpenTelemetryConfig struct {
	HTTP *OtelHTTP `yaml:"http,omitempty"`
	GRPC *OtelGPRC `yaml:"grpc,omitempty"`
}

type OtelHTTP struct {
	Headers  map[string]string `yaml:"headers,omitempty"`
	TLS      *types.ClientTLS  `yaml:"tls,omitempty"`
	Endpoint string            `yaml:"endpoint"`
}

type OtelGPRC struct {
	Headers  map[string]string `yaml:"headers,omitempty"`
	TLS      *types.ClientTLS  `yaml:"tls,omitempty"`
	Endpoint string            `yaml:"endpoint" default:"localhost:4317"`
	Insecure bool              `yaml:"insecure" default:"true"`
}

func (cfg TracingConfig) Validate() error {
	if cfg.SamplingRate > 1 || cfg.SamplingRate < 0 {
		return errors.New("invalid sampling rate")
	}
	return nil
}

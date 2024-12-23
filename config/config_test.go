package config

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedisConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 RedisConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: RedisConfig{
				Host:     "127.0.0.1",
				Port:     6379,
				Password: "",
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid port",
			cfg: RedisConfig{
				Host:     "127.0.0.1",
				Port:     65536,
				Password: "",
			},
			expectedValidateErr: errors.New("port must be in the range [0, 65535]"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestLogConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 LogConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: LogConfig{
				Level:  LogLevelInfo,
				Format: LogFormatText,
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid level",
			cfg: LogConfig{
				Level:  "",
				Format: LogFormatText,
			},
			expectedValidateErr: errors.New("invalid level: "),
		},
		{
			desc: "invalid level: x",
			cfg: LogConfig{
				Level:  "x",
				Format: LogFormatText,
			},
			expectedValidateErr: errors.New("invalid level: x"),
		},
		{
			desc: "invalid format",
			cfg: LogConfig{
				Level:  "info",
				Format: "",
			},
			expectedValidateErr: errors.New("invalid format: "),
		},
		{
			desc: "invalid format: x",
			cfg: LogConfig{
				Level:  "info",
				Format: "x",
			},
			expectedValidateErr: errors.New("invalid format: x"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestProxyConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 ProxyConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: ProxyConfig{
				Queue: Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: nil,
		},
		{
			desc: "max_request_body_size cannot be negative value",
			cfg: ProxyConfig{
				MaxRequestBodySize: -1,
				Queue: Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: errors.New("max_request_body_size cannot be negative value"),
		},
		{
			desc: "timeout_read cannot be negative value",
			cfg: ProxyConfig{
				TimeoutRead: -1,
				Queue: Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: errors.New("timeout_read cannot be negative value"),
		},
		{
			desc: "timeout_write cannot be negative value",
			cfg: ProxyConfig{
				TimeoutWrite: -1,
				Queue: Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: errors.New("timeout_write cannot be negative value"),
		},
		{
			desc: "invalid type: unknown",
			cfg: ProxyConfig{
				Queue: Queue{
					Type: "unknown",
				},
			},
			expectedValidateErr: errors.New("invalid queue: unknown type: unknown"),
		},
		{
			desc: "invalid queue",
			cfg: ProxyConfig{
				Queue: Queue{
					Type: "redis",
					Redis: RedisConfig{
						Port: 65536,
					},
				},
			},
			expectedValidateErr: errors.New("invalid queue: port must be in the range [0, 65535]"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestMetricsConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 MetricsConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: MetricsConfig{
				Attributes:   nil,
				Exports:      nil,
				PushInterval: 1,
				Opentelemetry: Opentelemetry{
					Protocol: "http/protobuf",
				},
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid export",
			cfg: MetricsConfig{
				Attributes:   nil,
				Exports:      []Export{"unknown"},
				PushInterval: 1,
				Opentelemetry: Opentelemetry{
					Protocol: "http/protobuf",
				},
			},
			expectedValidateErr: errors.New("invalid export: unknown"),
		},
		{
			desc: "invalid export",
			cfg: MetricsConfig{
				Attributes:   nil,
				Exports:      nil,
				PushInterval: 1,
				Opentelemetry: Opentelemetry{
					Protocol: "unknown",
				},
			},
			expectedValidateErr: errors.New("invalid protocol: unknown"),
		},
		{
			desc: "invalid PushInterval",
			cfg: MetricsConfig{
				Attributes:   nil,
				Exports:      nil,
				PushInterval: 61,
				Opentelemetry: Opentelemetry{
					Protocol: "http/protobuf",
				},
			},
			expectedValidateErr: errors.New("interval must be in the range [1, 60]"),
		},
	}

	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestTracingConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 TracingConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: TracingConfig{
				Enabled:      true,
				SamplingRate: 0,
				Opentelemetry: Opentelemetry{
					Protocol: "http/protobuf",
					Endpoint: "http://localhost:4318/v1/traces",
				},
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid sampling rate",
			cfg: TracingConfig{
				Enabled:      true,
				SamplingRate: 1.1,
				Opentelemetry: Opentelemetry{
					Protocol: "http/protobuf",
					Endpoint: "http://localhost:4318/v1/traces",
				},
			},
			expectedValidateErr: errors.New("sampling_rate must be in the range [0, 1]"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestConfig(t *testing.T) {
	cfg, err := Init()
	assert.Nil(t, err)
	assert.Nil(t, cfg.Validate())
}

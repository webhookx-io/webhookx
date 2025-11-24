package config

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config/modules"
)

func TestRedisConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 modules.RedisConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.RedisConfig{
				Host:     "127.0.0.1",
				Port:     6379,
				Password: "",
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid port",
			cfg: modules.RedisConfig{
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
		cfg                 modules.LogConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.LogConfig{
				Level:  modules.LogLevelInfo,
				Format: modules.LogFormatText,
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid level",
			cfg: modules.LogConfig{
				Level:  "",
				Format: modules.LogFormatText,
			},
			expectedValidateErr: errors.New("invalid level: "),
		},
		{
			desc: "invalid level: x",
			cfg: modules.LogConfig{
				Level:  "x",
				Format: modules.LogFormatText,
			},
			expectedValidateErr: errors.New("invalid level: x"),
		},
		{
			desc: "invalid format",
			cfg: modules.LogConfig{
				Level:  "info",
				Format: "",
			},
			expectedValidateErr: errors.New("invalid format: "),
		},
		{
			desc: "invalid format: x",
			cfg: modules.LogConfig{
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
		cfg                 modules.ProxyConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.ProxyConfig{
				Queue: modules.Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: nil,
		},
		{
			desc: "max_request_body_size cannot be negative value",
			cfg: modules.ProxyConfig{
				MaxRequestBodySize: -1,
				Queue: modules.Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: errors.New("max_request_body_size cannot be negative value"),
		},
		{
			desc: "timeout_read cannot be negative value",
			cfg: modules.ProxyConfig{
				TimeoutRead: -1,
				Queue: modules.Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: errors.New("timeout_read cannot be negative value"),
		},
		{
			desc: "timeout_write cannot be negative value",
			cfg: modules.ProxyConfig{
				TimeoutWrite: -1,
				Queue: modules.Queue{
					Type: "redis",
				},
			},
			expectedValidateErr: errors.New("timeout_write cannot be negative value"),
		},
		{
			desc: "invalid type: unknown",
			cfg: modules.ProxyConfig{
				Queue: modules.Queue{
					Type: "unknown",
				},
			},
			expectedValidateErr: errors.New("invalid queue: unknown type: unknown"),
		},
		{
			desc: "invalid queue",
			cfg: modules.ProxyConfig{
				Queue: modules.Queue{
					Type: "redis",
					Redis: modules.RedisConfig{
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
		cfg                 modules.MetricsConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.MetricsConfig{
				Attributes:   nil,
				Exports:      nil,
				PushInterval: 1,
				Opentelemetry: modules.OpentelemetryMetrics{
					Protocol: "http/protobuf",
				},
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid export",
			cfg: modules.MetricsConfig{
				Attributes:   nil,
				Exports:      []modules.Export{"unknown"},
				PushInterval: 1,
				Opentelemetry: modules.OpentelemetryMetrics{
					Protocol: "http/protobuf",
				},
			},
			expectedValidateErr: errors.New("invalid export: unknown"),
		},
		{
			desc: "invalid protocol",
			cfg: modules.MetricsConfig{
				Attributes:   nil,
				Exports:      nil,
				PushInterval: 1,
				Opentelemetry: modules.OpentelemetryMetrics{
					Protocol: "unknown",
				},
			},
			expectedValidateErr: errors.New("invalid protocol: unknown"),
		},
		{
			desc: "invalid PushInterval",
			cfg: modules.MetricsConfig{
				Attributes:   nil,
				Exports:      nil,
				PushInterval: 61,
				Opentelemetry: modules.OpentelemetryMetrics{
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
		cfg                 modules.TracingConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.TracingConfig{
				Enabled:      true,
				SamplingRate: 0,
				Opentelemetry: modules.OpentelemetryTracing{
					Protocol: "http/protobuf",
					Endpoint: "http://127.0.0.1:4318/v1/traces",
				},
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid sampling rate",
			cfg: modules.TracingConfig{
				Enabled:      true,
				SamplingRate: 1.1,
				Opentelemetry: modules.OpentelemetryTracing{
					Protocol: "http/protobuf",
					Endpoint: "http://127.0.0.1:4318/v1/traces",
				},
			},
			expectedValidateErr: errors.New("sampling_rate must be in the range [0, 1]"),
		},
		{
			desc: "invalid protocol",
			cfg: modules.TracingConfig{
				Opentelemetry: modules.OpentelemetryTracing{
					Protocol: "unknown",
				},
			},
			expectedValidateErr: errors.New("invalid protocol: unknown"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestAccessLogConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 modules.AccessLogConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.AccessLogConfig{
				File:   "/dev/stdout",
				Format: "text",
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid format",
			cfg: modules.AccessLogConfig{
				File:   "/dev/stdout",
				Format: "",
			},
			expectedValidateErr: errors.New("invalid format: "),
		},
		{
			desc: "invalid format: x",
			cfg: modules.AccessLogConfig{
				File:   "/dev/stdout",
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

func TestStatusConfig(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 modules.StatusConfig
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.StatusConfig{
				Listen:         "",
				DebugEndpoints: false,
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid listen",
			cfg: modules.StatusConfig{
				Listen:         "invalid",
				DebugEndpoints: true,
			},
			expectedValidateErr: errors.New("invalid listen 'invalid': address invalid: missing port in address"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
	}
}

func TestRole(t *testing.T) {
	cfg := New()

	cfg.Role = "standalone"
	assert.Nil(t, cfg.Validate())

	cfg.Role = "cp"
	assert.Nil(t, cfg.Validate())

	cfg.Role = "dp_worker"
	assert.Nil(t, cfg.Validate())

	cfg.Role = "dp_proxy"
	assert.Nil(t, cfg.Validate())

	cfg.Role = ""
	assert.Equal(t, errors.New("invalid role: ''"), cfg.Validate())
}

func TestWorkerConfig(t *testing.T) {
	tests := []struct {
		desc        string
		cfg         modules.WorkerConfig
		validateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.WorkerConfig{
				Enabled: false,
				Deliverer: modules.WorkerDeliverer{
					Timeout: 0,
					ACL: modules.ACLConfig{
						Deny: []string{"@default", "0.0.0.0", "0.0.0.0/32", "*.example.com", "foo.example.com", "::1/128"},
					},
				},
				Pool: modules.Pool{},
			},
			validateErr: nil,
		},
		{
			desc: "invalid deliverer configuration: negative timeout",
			cfg: modules.WorkerConfig{
				Deliverer: modules.WorkerDeliverer{
					Timeout: -1,
					ACL:     modules.ACLConfig{},
				},
			},
			validateErr: errors.New("deliverer.timeout cannot be negative"),
		},
		{
			desc: "invalid deliverer configuration: invalid acl configuration 1",
			cfg: modules.WorkerConfig{
				Deliverer: modules.WorkerDeliverer{
					Timeout: 0,
					ACL: modules.ACLConfig{
						Deny: []string{"default"},
					},
				},
			},
			validateErr: errors.New("invalid rule 'default': requires IP, CIDR, hostname, or pre-configured name"),
		},
		{
			desc: "invalid deliverer configuration: invalid acl configuration 2",
			cfg: modules.WorkerConfig{
				Deliverer: modules.WorkerDeliverer{
					Timeout: 0,
					ACL: modules.ACLConfig{
						Deny: []string{"*"},
					},
				},
			},
			validateErr: errors.New("invalid rule '*': requires IP, CIDR, hostname, or pre-configured name"),
		},
		{
			desc: "invalid deliverer configuration: unicode hostname",
			cfg: modules.WorkerConfig{
				Deliverer: modules.WorkerDeliverer{
					Timeout: 0,
					ACL: modules.ACLConfig{
						Deny: []string{"тест.example.com"},
					},
				},
			},
			validateErr: errors.New("invalid rule 'тест.example.com': requires IP, CIDR, hostname, or pre-configured name"),
		},
	}
	for _, test := range tests {
		actual := test.cfg.Validate()
		assert.Equal(t, test.validateErr, actual, "expected %v got %v", test.validateErr, actual)
	}
}

func TestWorkerProxyConfig(t *testing.T) {
	tests := []struct {
		desc        string
		cfg         modules.WorkerDeliverer
		validateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.WorkerDeliverer{
				Proxy: "http://example.com:8080",
			},
			validateErr: nil,
		},
		{
			desc: "invalid proxy url: missing schema",
			cfg: modules.WorkerDeliverer{
				Proxy: "example.com",
			},
			validateErr: errors.New("invalid proxy url: 'example.com'"),
		},
		{
			desc: "invalid proxy url: invalid schema ",
			cfg: modules.WorkerDeliverer{
				Proxy: "ftp://example.com",
			},
			validateErr: errors.New("proxy schema must be http or https"),
		},
		{
			desc: "invalid proxy url: missing host ",
			cfg: modules.WorkerDeliverer{
				Proxy: "http://",
			},
			validateErr: errors.New("invalid proxy url: 'http://'"),
		},
		{
			desc: "invalid proxy url: missing host ",
			cfg: modules.WorkerDeliverer{
				Proxy: "http ://",
			},
			validateErr: errors.New("invalid proxy url: parse \"http ://\": first path segment in URL cannot contain colon"),
		},
	}
	for _, test := range tests {
		actual := test.cfg.Validate()
		assert.Equal(t, test.validateErr, actual, "expected %v got %v", test.validateErr, actual)
	}
}

func TestSecretConfig(t *testing.T) {
	tests := []struct {
		desc        string
		cfg         modules.SecretConfig
		validateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.SecretConfig{
				Vault: modules.VaultProvider{
					AuthMethod: "token",
				},
			},
			validateErr: nil,
		},
		{
			desc: "invalid provider",
			cfg: modules.SecretConfig{
				Providers: []modules.Provider{"aws", "vault", "unknown"},
				Vault: modules.VaultProvider{
					AuthMethod: "token",
				},
			},
			validateErr: errors.New("invalid provider: unknown"),
		},
		{
			desc: "invalid vault.auth_method",
			cfg: modules.SecretConfig{
				Vault: modules.VaultProvider{
					AuthMethod: "unknown",
				},
			},
			validateErr: errors.New("invalid auth_method: unknown"),
		},
	}
	for _, test := range tests {
		actual := test.cfg.Validate()
		assert.Equal(t, test.validateErr, actual, "expected %v got %v", test.validateErr, actual)
	}
}

func TestConfig(t *testing.T) {
	cfg := New()
	assert.Nil(t, cfg.Validate())
	str := cfg.String()
	cfg2 := &Config{}
	err := json.Unmarshal([]byte(str), cfg2)
	// restore password
	cfg2.Database.Password = cfg.Database.Password
	cfg2.Redis.Password = cfg.Redis.Password
	cfg2.Proxy.Queue.Redis.Password = cfg.Proxy.Queue.Redis.Password
	assert.Nil(t, err)
	assert.Equal(t, cfg, cfg2)
}

func TestInitWithFile(t *testing.T) {
	cfg := New()
	err := Load("./testdata/config-empty.yml", cfg)
	assert.Nil(t, err)
	assert.Nil(t, cfg.Validate())
}

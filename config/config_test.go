package config

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/config/modules"
	configtypes "github.com/webhookx-io/webhookx/config/types"
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
		{
			desc: "invalid instrumentations",
			cfg: modules.TracingConfig{
				Instrumentations: []string{"unknown"},
				Opentelemetry: modules.OpentelemetryTracing{
					Protocol: "http/protobuf",
					Endpoint: "http://127.0.0.1:4318/v1/traces",
				},
			},
			expectedValidateErr: errors.New("invalid instrumentations: unknown"),
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
				CircuitBreaker: modules.CircuitBreaker{
					WindowSize:              3600,
					FailureRateThreshold:    80,
					MinimumRequestThreshold: 100,
				},
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

func TestCircuitBreaker(t *testing.T) {
	tests := []struct {
		desc                string
		cfg                 modules.CircuitBreaker
		expectedValidateErr error
	}{
		{
			desc: "sanity",
			cfg: modules.CircuitBreaker{
				WindowSize:              3600,
				FailureRateThreshold:    80,
				MinimumRequestThreshold: 100,
			},
			expectedValidateErr: nil,
		},
		{
			desc: "invalid sliding_window: 0",
			cfg: modules.CircuitBreaker{
				WindowSize:              0,
				FailureRateThreshold:    80,
				MinimumRequestThreshold: 100,
			},
			expectedValidateErr: errors.New("window_size must be in the range [60, 86400]"),
		},
		{
			desc: "invalid sliding_window: 86401",
			cfg: modules.CircuitBreaker{
				WindowSize:              86401,
				FailureRateThreshold:    80,
				MinimumRequestThreshold: 100,
			},
			expectedValidateErr: errors.New("window_size must be in the range [60, 86400]"),
		},
		{
			desc: "invalid failure_rate_threshold: 0",
			cfg: modules.CircuitBreaker{
				WindowSize:              3600,
				FailureRateThreshold:    0,
				MinimumRequestThreshold: 100,
			},
			expectedValidateErr: errors.New("failure_rate_threshold must be in the range [1, 100]"),
		},
		{
			desc: "invalid failure_rate_threshold: 101",
			cfg: modules.CircuitBreaker{
				WindowSize:              3600,
				FailureRateThreshold:    101,
				MinimumRequestThreshold: 100,
			},
			expectedValidateErr: errors.New("failure_rate_threshold must be in the range [1, 100]"),
		},
		{
			desc: "invalid minimum_request_threshold: 0",
			cfg: modules.CircuitBreaker{
				WindowSize:              3600,
				FailureRateThreshold:    80,
				MinimumRequestThreshold: 0,
			},
			expectedValidateErr: errors.New("minimum_request_threshold must be greater than 1"),
		},
	}
	for _, test := range tests {
		actualValidateErr := test.cfg.Validate()
		assert.Equal(t, test.expectedValidateErr, actualValidateErr, "expected %v got %v", test.expectedValidateErr, actualValidateErr)
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
				Providers: []string{"aws", "vault", "unknown"},
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

func TestRetentionConfig(t *testing.T) {
	tests := []struct {
		desc        string
		cfg         modules.RetentionConfig
		validateErr error
	}{
		{
			desc: "valid disable",
			cfg: modules.RetentionConfig{
				Enabled:  false,
				Interval: configtypes.Duration(24 * time.Hour),
			},
			validateErr: nil,
		},
		{
			desc: "valid enabled with ttls",
			cfg: modules.RetentionConfig{
				Enabled:  true,
				Interval: configtypes.Duration(time.Hour),
				TTL: modules.RetentionTTLConfig{
					Events:   configtypes.Duration(24 * time.Hour),
					Attempts: configtypes.Duration(48 * time.Hour),
				},
			},
			validateErr: nil,
		},
		{
			desc: "interval less than 1g",
			cfg: modules.RetentionConfig{
				Interval: configtypes.Duration(30 * time.Minute),
			},
			validateErr: errors.New("minimum interval is 1h"),
		},
		{
			desc: "negative events ttl",
			cfg: modules.RetentionConfig{
				Interval: configtypes.Duration(time.Hour),
				TTL: modules.RetentionTTLConfig{
					Events: configtypes.Duration(-time.Second),
				},
			},
			validateErr: errors.New("ttl.events cannot be negative"),
		},
		{
			desc: "events ttl less than 1d",
			cfg: modules.RetentionConfig{
				Interval: configtypes.Duration(time.Hour),
				TTL: modules.RetentionTTLConfig{
					Events: configtypes.Duration(12 * time.Hour),
				},
			},
			validateErr: errors.New("minimum ttl.events is 1d"),
		},
		{
			desc: "negative attempts ttl",
			cfg: modules.RetentionConfig{
				Interval: configtypes.Duration(time.Hour),
				TTL: modules.RetentionTTLConfig{
					Attempts: configtypes.Duration(-time.Second),
				},
			},
			validateErr: errors.New("ttl.attempts cannot be negative"),
		},
		{
			desc: "attempts ttl less than 1d",
			cfg: modules.RetentionConfig{
				Interval: configtypes.Duration(time.Hour),
				TTL: modules.RetentionTTLConfig{
					Attempts: configtypes.Duration(12 * time.Hour),
				},
			},
			validateErr: errors.New("minimum ttl.attempts is 1d"),
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

func TestLoadRetentionConfig(t *testing.T) {
	cfg := New()
	err := NewLoader(cfg).
		WithEnvPrefix("WEBHOOKX").
		WithEnv(map[string]string{}).
		WithFileContent([]byte(`
retention:
  enabled: true
  ttl:
    events: 30d
    attempts: 60d
`)).
		Load()

	assert.NoError(t, err)
	assert.True(t, cfg.Retention.Enabled)
	assert.Equal(t, configtypes.Duration(30*24*time.Hour), cfg.Retention.TTL.Events)
	assert.Equal(t, configtypes.Duration(60*24*time.Hour), cfg.Retention.TTL.Attempts)
	assert.NoError(t, cfg.Validate())

	data, err := json.Marshal(cfg.Retention)
	assert.NoError(t, err)
	var decoded modules.RetentionConfig
	assert.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, cfg.Retention, decoded)
}

func TestLoadRetentionConfigFromEnvironment(t *testing.T) {
	cfg := New()
	err := NewLoader(cfg).
		WithEnvPrefix("WEBHOOKX").
		WithEnv(map[string]string{
			"WEBHOOKX_RETENTION_ENABLED":      "true",
			"WEBHOOKX_RETENTION_TTL_EVENTS":   "30d",
			"WEBHOOKX_RETENTION_TTL_ATTEMPTS": "60d",
		}).
		Load()

	assert.NoError(t, err)
	assert.True(t, cfg.Retention.Enabled)
	assert.Equal(t, configtypes.Duration(30*24*time.Hour), cfg.Retention.TTL.Events)
	assert.Equal(t, configtypes.Duration(60*24*time.Hour), cfg.Retention.TTL.Attempts)
}

func TestLoadRetentionConfigRejectsInvalidTTL(t *testing.T) {
	cfg := New()
	err := NewLoader(cfg).
		WithEnv(map[string]string{}).
		WithFileContent([]byte(`
retention:
  ttl:
    events: 30days
`)).
		Load()

	assert.Error(t, err)
}

func TestRetentionConfigInterval(t *testing.T) {
	for _, interval := range []time.Duration{
		-time.Second,
		0,
		time.Hour - time.Nanosecond,
	} {
		t.Run(interval.String(), func(t *testing.T) {
			cfg := modules.RetentionConfig{
				Enabled:  true,
				Interval: configtypes.Duration(interval),
			}

			assert.EqualError(t, cfg.Validate(), "minimum interval is 1h")
		})
	}

	cfg := modules.RetentionConfig{
		Enabled:  true,
		Interval: configtypes.Duration(time.Hour),
	}
	assert.NoError(t, cfg.Validate())
}

func TestDurationJSONRoundTrip(t *testing.T) {
	values := []configtypes.Duration{
		configtypes.Duration(30 * 24 * time.Hour),
		configtypes.Duration(-30 * 24 * time.Hour),
		configtypes.Duration(-1 << 63),
		configtypes.Duration(1<<63 - 1),
	}

	for _, expected := range values {
		t.Run(expected.String(), func(t *testing.T) {
			data, err := json.Marshal(expected)
			assert.NoError(t, err)

			var actual configtypes.Duration
			assert.NoError(t, json.Unmarshal(data, &actual))
			assert.Equal(t, expected, actual)
		})
	}
}

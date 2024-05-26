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

func TestConfig(t *testing.T) {
	cfg := cfg
	assert.Nil(t, cfg.Validate())
}

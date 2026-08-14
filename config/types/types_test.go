package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_Duration(t *testing.T) {
	tests := []struct {
		name     string
		d        Duration
		expected time.Duration
	}{
		{
			name:     "positive duration",
			d:        Duration(10 * time.Second),
			expected: 10 * time.Second,
		},
		{
			name:     "zero duration",
			d:        Duration(0),
			expected: 0,
		},
		{
			name:     "negative duration",
			d:        Duration(-5 * time.Minute),
			expected: -5 * time.Minute,
		},
		{
			name:     "days duration",
			d:        Duration(48 * time.Hour),
			expected: 48 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.d.Duration())
		})
	}
}

func TestDuration_String(t *testing.T) {
	tests := []struct {
		name     string
		d        Duration
		expected string
	}{
		{
			name:     "zero",
			d:        Duration(0),
			expected: "0",
		},
		{
			name:     "positive seconds",
			d:        Duration(10 * time.Second),
			expected: "10s",
		},
		{
			name:     "positive minutes and seconds",
			d:        Duration(1*time.Minute + 30*time.Second),
			expected: "1m30s",
		},
		{
			name:     "positive hours",
			d:        Duration(2 * time.Hour),
			expected: "2h",
		},
		{
			name:     "positive days",
			d:        Duration(24 * time.Hour),
			expected: "1d",
		},
		{
			name:     "positive days and hours",
			d:        Duration(25 * time.Hour),
			expected: "1d1h",
		},
		{
			name:     "positive milliseconds",
			d:        Duration(500 * time.Millisecond),
			expected: "500ms",
		},
		{
			name:     "negative duration",
			d:        Duration(-10 * time.Second),
			expected: "-10s",
		},
		{
			name:     "negative hours",
			d:        Duration(-24 * time.Hour),
			expected: "-24h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.d.String())
		})
	}
}

func TestDuration_Decode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Duration
		expectErr bool
	}{
		{
			name:      "zero string",
			input:     "0",
			expected:  Duration(0),
			expectErr: false,
		},
		{
			name:      "zero with unit",
			input:     "0s",
			expected:  Duration(0),
			expectErr: false,
		},
		{
			name:      "seconds",
			input:     "30s",
			expected:  Duration(30 * time.Second),
			expectErr: false,
		},
		{
			name:      "minutes",
			input:     "15m",
			expected:  Duration(15 * time.Minute),
			expectErr: false,
		},
		{
			name:      "hours",
			input:     "2h",
			expected:  Duration(2 * time.Hour),
			expectErr: false,
		},
		{
			name:      "days",
			input:     "3d",
			expected:  Duration(72 * time.Hour),
			expectErr: false,
		},
		{
			name:      "fractional days",
			input:     "1.5d",
			expected:  Duration(36 * time.Hour),
			expectErr: false,
		},
		{
			name:      "milliseconds",
			input:     "500ms",
			expected:  Duration(500 * time.Millisecond),
			expectErr: false,
		},
		{
			name:      "combined day and hour",
			input:     "1d12h",
			expected:  Duration(36 * time.Hour),
			expectErr: false,
		},
		{
			name:      "negative duration",
			input:     "-1h",
			expected:  Duration(-1 * time.Hour),
			expectErr: false,
		},
		{
			name:      "invalid format - plain letters",
			input:     "invalid",
			expectErr: true,
		},
		{
			name:      "invalid unit",
			input:     "10x",
			expectErr: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.Decode(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		d        Duration
		expected string
	}{
		{
			name:     "positive duration",
			d:        Duration(10 * time.Second),
			expected: `"10s"`,
		},
		{
			name:     "days duration",
			d:        Duration(24 * time.Hour),
			expected: `"1d"`,
		},
		{
			name:     "zero duration",
			d:        Duration(0),
			expected: `"0"`,
		},
		{
			name:     "negative duration",
			d:        Duration(-10 * time.Second),
			expected: `"-10s"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.d)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, string(data))
		})
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		expected  Duration
		expectErr bool
	}{
		{
			name:      "valid seconds",
			jsonInput: `"10s"`,
			expected:  Duration(10 * time.Second),
			expectErr: false,
		},
		{
			name:      "valid days",
			jsonInput: `"2d"`,
			expected:  Duration(48 * time.Hour),
			expectErr: false,
		},
		{
			name:      "valid zero",
			jsonInput: `"0"`,
			expected:  Duration(0),
			expectErr: false,
		},
		{
			name:      "valid negative",
			jsonInput: `"-1h"`,
			expected:  Duration(-time.Hour),
			expectErr: false,
		},
		{
			name:      "invalid json - number type",
			jsonInput: `100`,
			expectErr: true,
		},
		{
			name:      "invalid json - object type",
			jsonInput: `{"time": "10s"}`,
			expectErr: true,
		},
		{
			name:      "invalid json - boolean type",
			jsonInput: `true`,
			expectErr: true,
		},
		{
			name:      "invalid duration text",
			jsonInput: `"not-a-duration"`,
			expectErr: true,
		},
		{
			name:      "malformed json",
			jsonInput: `"unclosed`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := json.Unmarshal([]byte(tt.jsonInput), &d)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestDuration_JSONRoundTrip(t *testing.T) {
	type Config struct {
		Timeout  Duration `json:"timeout"`
		Interval Duration `json:"interval"`
	}

	cfg := Config{
		Timeout:  Duration(30 * time.Second),
		Interval: Duration(24 * time.Hour),
	}

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var parsed Config
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, cfg, parsed)
}

func TestDuration_MarshalText(t *testing.T) {
	d := Duration(5 * time.Minute)
	data, err := d.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, "5m", string(data))
}

func TestDuration_UnmarshalText(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Duration
		expectErr bool
	}{
		{
			name:      "valid text",
			input:     "1d",
			expected:  Duration(24 * time.Hour),
			expectErr: false,
		},
		{
			name:      "invalid text",
			input:     "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalText([]byte(tt.input))
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}

func TestMap_Decode(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  Map
		expectErr bool
	}{
		{
			name:      "valid json map",
			input:     `{"k1":"v1","k2":"v2"}`,
			expected:  Map{"k1": "v1", "k2": "v2"},
			expectErr: false,
		},
		{
			name:      "empty json map",
			input:     `{}`,
			expected:  Map{},
			expectErr: false,
		},
		{
			name:      "invalid json",
			input:     `invalid`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Map
			err := m.Decode(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, m)
			}
		})
	}
}

func TestPassword_MarshalJSON(t *testing.T) {
	p := Password("secret_password_123")
	data, err := json.Marshal(p)
	assert.NoError(t, err)
	assert.Equal(t, `"******"`, string(data))
}

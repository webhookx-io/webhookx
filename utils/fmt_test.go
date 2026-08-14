package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "1d", FormatDuration(time.Hour*24))
	assert.Equal(t, "1h", FormatDuration(time.Hour))
	assert.Equal(t, "1m", FormatDuration(time.Minute))
	assert.Equal(t, "1s", FormatDuration(time.Second))
	assert.Equal(t, "0", FormatDuration(0))
	assert.Equal(t, "1d2h3m4s", FormatDuration(24*time.Hour+2*time.Hour+3*time.Minute+4*time.Second))
	assert.Equal(t, "1ms", FormatDuration(time.Millisecond))
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		value    string
		expected time.Duration
	}{
		{value: "30d", expected: 30 * 24 * time.Hour},
		{value: "1d2h3m4s", expected: 26*time.Hour + 3*time.Minute + 4*time.Second},
		{value: "1.5d", expected: 36 * time.Hour},
		{value: "90m", expected: 90 * time.Minute},
		{value: "-1d2h", expected: -26 * time.Hour},
	}

	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			actual, err := ParseDuration(test.value)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestParseDurationRejectsInvalidValue(t *testing.T) {
	_, err := ParseDuration("30days")
	assert.Error(t, err)
}

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
	assert.Equal(t, "0s", FormatDuration(0))
	assert.Equal(t, "1d2h3m4s", FormatDuration(24*time.Hour+2*time.Hour+3*time.Minute+4*time.Second))
	assert.Equal(t, "1ms", FormatDuration(time.Millisecond))
}

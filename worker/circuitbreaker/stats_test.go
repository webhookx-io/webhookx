package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot(t *testing.T) {
	now := time.Now()
	s := Stats{
		StartTime: now.Unix(),
		EndTime:   now.Unix(),
		Success:   9,
		Failure:   1,
	}
	assert.EqualValues(t, 10, s.TotalRequest())
	assert.EqualValues(t, 9, s.TotalSuccess())
	assert.EqualValues(t, 1, s.TotalFailures())
	assert.Equal(t, 0.1, s.FailureRate())
}

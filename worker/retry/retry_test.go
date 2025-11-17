package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	r1 := NewRetry(FixedStrategy)
	assert.NotNil(t, r1)
}

func TestFixedRetry(t *testing.T) {
	r := NewRetry(FixedStrategy)
	assert.Equal(t, Stop, r.NextDelay(1))
}

func TestFixedRetryWithOptions(t *testing.T) {
	r := NewRetry(FixedStrategy, WithFixedDelay([]int64{1, 2, 3, 4}))
	assert.Equal(t, time.Second*1, r.NextDelay(1))
	assert.Equal(t, time.Second*2, r.NextDelay(2))
	assert.Equal(t, time.Second*3, r.NextDelay(3))
	assert.Equal(t, time.Second*4, r.NextDelay(4))
	assert.Equal(t, Stop, r.NextDelay(5))
}

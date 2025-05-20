package loglimiter

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test(t *testing.T) {
	limiter := NewLimiter(time.Millisecond * 100)

	n := 0
	ticker := time.NewTicker(time.Millisecond * 10)
	timeout := time.NewTimer(time.Millisecond * 1001)
	defer timeout.Stop()
	for {
		select {
		case <-ticker.C:
			if limiter.Allow("key") {
				n++
			}
		case <-timeout.C:
			assert.Equal(t, 10, n)
			return
		}
	}

}

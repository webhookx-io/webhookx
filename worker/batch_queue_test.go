package worker

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	t.Run("should call flush function when BatchSize is exceeded", func(t *testing.T) {
		q := NewBatchQueue[string](100, 10, time.Second)
		var n atomic.Int32
		q.Consume(func(list []string) { n.Store(int32(len(list))) })
		for i := 1; i <= 10; i++ {
			q.Add(strconv.Itoa(i))
		}
		q.Close()
		assert.EqualValues(t, 10, n.Load())
	})
}

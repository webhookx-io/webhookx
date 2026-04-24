package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	bucket := UnixBucket{}
	now := time.Now().Unix()
	bucket.Record(now, Success)
	bucket.Record(now, Error)
	s := bucket.Snapshot()
	assert.EqualValues(t, s.StartTime, now)
	assert.EqualValues(t, s.EndTime, now)
	assert.EqualValues(t, 1, s.Success)
	assert.EqualValues(t, 1, s.Failure)

	bucket.Record(now-1, Success) // should be discarded
	bucket.Record(now-1, Error)   // should be discarded
	s = bucket.Snapshot()

	assert.EqualValues(t, s.StartTime, now)
	assert.EqualValues(t, s.EndTime, now)
	assert.EqualValues(t, 1, s.Success)
	assert.EqualValues(t, 1, s.Failure)
}

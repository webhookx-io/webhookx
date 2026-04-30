package metrics

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnixBucket(t *testing.T) {
	bucket := UnixAggregation{}
	now := time.Now().Unix()
	bucket.Record(now, Success)
	bucket.Record(now, Error)
	s := bucket.Snapshot()
	assert.EqualValues(t, s.Start(), now)
	assert.EqualValues(t, s.Until(), now + 1)
	assert.EqualValues(t, 1, s.SuccessCount())
	assert.EqualValues(t, 1, s.ErrorCount())

	bucket.Record(now-1, Success) // should be discarded
	bucket.Record(now-1, Error)   // should be discarded
	s = bucket.Snapshot()

	assert.EqualValues(t, s.Start(), now)
	assert.EqualValues(t, s.Until(), now + 1)
	assert.EqualValues(t, 1, s.SuccessCount())
	assert.EqualValues(t, 1, s.ErrorCount())
}


func TestSnapshot(t *testing.T) {
	now := time.Now()
	s := snapshot{
		start: now.Unix(),
		until:   now.Unix(),
		success:   9,
		error:     1,
	}
	assert.EqualValues(t, 9, s.SuccessCount())
	assert.EqualValues(t, 1, s.ErrorCount())
	assert.EqualValues(t, now.Unix(), s.start)
	assert.EqualValues(t, now.Unix(), s.until)
}


type MutexUnixBucket struct {
	mutex   sync.Mutex
	unix    int64
	success int64
	error   int64
}

func (b *MutexUnixBucket) Record(timestamp int64, event Event) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.unix = timestamp
	switch event {
	case Success:
		b.success++
	case Error:
		b.error++
	}
}

func (b *MutexUnixBucket) Snapshot() Snapshot {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return snapshot{
		start:   b.unix,
		until:   b.unix + 1,
		success: b.success,
		error:   b.error,
	}
}

func BenchmarkMutexUnixBucket(b *testing.B) {
	bucket := MutexUnixBucket{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bucket.Record(time.Now().UnixMilli(), Success)
		}
	})
	fmt.Println(bucket.Snapshot())
}

func BenchmarkUnixBucket(b *testing.B) {
	bucket := UnixAggregation{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bucket.Record(time.Now().UnixMilli(), Success)
		}
	})
	fmt.Println(bucket.Snapshot())
}

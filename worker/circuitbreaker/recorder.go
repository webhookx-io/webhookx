package circuitbreaker

import (
	"sync/atomic"
	"time"
)

const (
	defaultBufferSize int64 = 60
)

type Recorder struct {
	size   int64
	buffer []UnixBucket

	lastSync atomic.Int64
}

func NewRecorder() *Recorder {
	cb := &Recorder{
		size:   defaultBufferSize,
		buffer: make([]UnixBucket, defaultBufferSize),
	}
	return cb
}

func (r *Recorder) Record(timestamp int64, outcome Outcome) {
	bucket := r.bucket(timestamp)
	bucket.Record(timestamp, outcome)
}

func (r *Recorder) bucket(ts int64) *UnixBucket {
	index := ts % r.size
	return &r.buffer[index]
}

func (r *Recorder) LastSync() int64 {
	return r.lastSync.Load()
}

func (r *Recorder) SetLastSync(ts int64) {
	r.lastSync.Store(ts)
}

func (r *Recorder) Aggregate(unit time.Duration, from, to int64) (stats []Stats) {
	unitSeconds := int64(unit.Seconds())

	aggregated := make(map[int64]Stats)

	for ts := from; ts <= to; ts++ {
		s := r.bucket(ts).Snapshot()

		if s.StartTime != ts {
			continue
		}

		key := (ts / unitSeconds) * unitSeconds
		tmp := aggregated[key]
		tmp.Success += s.TotalSuccess()
		tmp.Failure += s.TotalFailures()
		aggregated[key] = tmp
	}

	for timestamp, m := range aggregated {
		stats = append(stats, Stats{
			StartTime: timestamp,
			EndTime:   timestamp + unitSeconds,
			Success:   m.Success,
			Failure:   m.Failure,
		})
	}

	return
}

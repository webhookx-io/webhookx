package circuitbreaker

import (
	"maps"
	"slices"
	"sync/atomic"

	"github.com/webhookx-io/webhookx/worker/circuitbreaker/metrics"
)

type Unit int64

func (u Unit) Seconds() int64 { return int64(u) }

const (
	Minute Unit = 60
	Hour        = 60 * Minute
)

type Recorder struct {
	size     int64
	buffers  []metrics.Aggregator
	lastSync atomic.Int64
}

func NewRecorder(size int64) *Recorder {
	cb := &Recorder{
		size:    size,
		buffers: make([]metrics.Aggregator, size),
	}
	for i := range cb.buffers {
		cb.buffers[i] = &metrics.UnixAggregation{}
	}
	return cb
}

func (r *Recorder) Record(timestamp int64, event metrics.Event) {
	bucket := r.bucket(timestamp)
	bucket.Record(timestamp, event)
}

func (r *Recorder) bucket(ts int64) metrics.Aggregator {
	index := ts % r.size
	return r.buffers[index]
}

func (r *Recorder) LastSync() int64 {
	return r.lastSync.Load()
}

func (r *Recorder) SetLastSync(ts int64) {
	r.lastSync.Store(ts)
}

func (r *Recorder) Aggregate(unit Unit, from, to int64) []TimeBucketMetric {
	metricCache := make(map[int64]TimeBucketMetric)
	for _, buffer := range r.buffers {
		s := buffer.Snapshot()
		ts := s.Start()
		if ts < from || ts > to {
			continue
		}

		key := ts - (ts % unit.Seconds())
		m := metricCache[key]
		m.Success += s.SuccessCount()
		m.Error += s.ErrorCount()
		metricCache[key] = m
	}

	keys := slices.Collect(maps.Keys(metricCache))
	slices.Sort(keys)

	list := make([]TimeBucketMetric, len(metricCache))
	for i, key := range keys {
		m := metricCache[key]
		m.Start = key
		m.Until = key + unit.Seconds()
		list[i] = m
	}

	return list
}

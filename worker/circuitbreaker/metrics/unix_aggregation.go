package metrics

import (
	"runtime"
	"sync/atomic"
)

const updating = -1

// UnixAggregation is a lock-free and concurrent-safe aggregator
type UnixAggregation struct {
	unix    atomic.Int64
	success atomic.Int64
	error   atomic.Int64
	_       [40]byte
}

func (b *UnixAggregation) Record(timestamp int64, event Event) {
	var success, error int64
	switch event {
	case Success:
		success = 1
	case Error:
		error = 1
	}

	retried := false
	for {
		unix := b.unix.Load()
		if unix == timestamp {
			b.add(success, error)
			return
		}

		if unix > timestamp {
			return // discard
		}

		if unix != updating && b.unix.CompareAndSwap(unix, updating) {
			b.success.Store(success)
			b.error.Store(error)
			b.unix.Store(timestamp) // reset finish
			return
		}

		if !retried {
			retried = true
			continue
		}

		runtime.Gosched()
	}
}

func (b *UnixAggregation) add(success, error int64) {
	if success > 0 {
		b.success.Add(success)
	}
	if error > 0 {
		b.error.Add(error)
	}
}

func (b *UnixAggregation) Snapshot() Snapshot {
	unix := b.unix.Load()
	for unix == updating {
		unix = b.unix.Load()
	}

	return snapshot{
		start:   unix,
		until:   unix + 1,
		success: b.success.Load(),
		error:   b.error.Load(),
	}
}

type snapshot struct {
	start   int64
	until   int64
	success int64
	error   int64
}

func (s snapshot) Start() int64 {
	return s.start
}

func (s snapshot) Until() int64 {
	return s.until
}

func (s snapshot) SuccessCount() int64 {
	return s.success
}

func (s snapshot) ErrorCount() int64 {
	return s.error
}

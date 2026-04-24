package circuitbreaker

import (
	"runtime"
	"sync/atomic"
)

const resetFlag = -1

type UnixBucket struct {
	unix    atomic.Int64
	success atomic.Int64
	failure atomic.Int64
	_       [40]byte
}

func (b *UnixBucket) Record(timestamp int64, outcome Outcome) {
	var success, failure int64
	switch outcome {
	case Success:
		success = 1
	case Error:
		failure = 1
	}

	retried := false
	for {
		unix := b.unix.Load()
		if unix == timestamp {
			b.success.Add(success)
			b.failure.Add(failure)
			return
		}

		if unix > timestamp {
			return // discard
		}

		if unix != resetFlag && b.unix.CompareAndSwap(unix, resetFlag) {
			b.success.Store(success)
			b.failure.Store(failure)
			b.unix.Store(timestamp) // reset done
			return
		}

		if !retried {
			retried = true
			continue
		}

		runtime.Gosched()
	}
}

func (b *UnixBucket) Snapshot() Stats {
	unix := b.unix.Load()
	for unix == resetFlag {
		unix = b.unix.Load()
	}

	return Stats{
		StartTime: unix,
		EndTime:   unix,
		Success:   b.success.Load(),
		Failure:   b.failure.Load(),
	}
}

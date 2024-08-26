package retry

import "time"

type FixedStrategyRetry struct {
	fixedDelaySeconds []int64
}

func newFixedStrategyRetry() *FixedStrategyRetry {
	return &FixedStrategyRetry{}
}

func WithFixedDelay(fixedDelaySeconds []int64) Option {
	return func(r Retry) {
		retry := r.(*FixedStrategyRetry)
		retry.fixedDelaySeconds = fixedDelaySeconds
	}
}

func (r *FixedStrategyRetry) NextDelay(attempts int) time.Duration {
	if attempts > len(r.fixedDelaySeconds) {
		return Stop
	}
	seconds := r.fixedDelaySeconds[attempts-1]
	return time.Duration(seconds) * time.Second
}

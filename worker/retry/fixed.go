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

func (r *FixedStrategyRetry) NextDelay(attemps int) time.Duration {
	if attemps > len(r.fixedDelaySeconds) {
		return Stop
	}
	seconds := r.fixedDelaySeconds[attemps-1]
	return time.Duration(seconds) * time.Second
}

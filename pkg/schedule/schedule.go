package schedule

import (
	"context"
	"time"
)

func ScheduleWithoutDelay(ctx context.Context, fn func(), interval time.Duration) {
	Schedule(ctx, fn, interval, 0)
}

func Schedule(ctx context.Context, fn func(), interval time.Duration, delay time.Duration) {
	go func() {
		if delay > 0 {
			time.Sleep(delay)
		}
		fn()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fn()
			}
		}
	}()
}

package schedule

import (
	"context"
	"time"
)

func Schedule(ctx context.Context, fn func(), interval time.Duration) {
	go func() {
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

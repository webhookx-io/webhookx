package safe

import "go.uber.org/zap"

func Go(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				zap.S().Errorf("goroutine panic: %v", err)
			}
		}()
		fn()
	}()
}

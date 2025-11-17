package safe

import (
	"runtime"

	"go.uber.org/zap"
)

func Go(fn func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				zap.S().Errorf("goroutine panic: %v\n %s", err, buf)
			}
		}()
		fn()
	}()
}

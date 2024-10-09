package pool

import (
	"fmt"
	"runtime"
)

type Task interface {
	Execute()
}

type task struct {
	fn func()
}

func (t *task) Execute() {
	defer func() {
		if e := recover(); e != nil {
			buf := make([]byte, 2048)
			n := runtime.Stack(buf, false)
			buf = buf[:n]
			fmt.Printf("panic recovered: %v\n %s\n", e, buf)
		}
	}()
	t.fn()
}

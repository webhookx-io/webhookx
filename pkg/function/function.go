package function

import (
	"time"
)

const Timeout = time.Second * 1

type Function interface {
	Execute(ctx *ExecutionContext) (ExecutionResult, error)
}

func New(language string, script string) Function {
	if language == "javascript" {
		return NewJavaScript(script)
	}
	return nil
}

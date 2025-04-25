package function

import (
	"github.com/webhookx-io/webhookx/plugins/function/function/javascript"
	"github.com/webhookx-io/webhookx/plugins/function/sdk"
	"time"
)

type Function interface {
	Execute(ctx *sdk.ExecutionContext) (sdk.ExecutionResult, error)
}

func New(language string, script string) Function {
	if language == "javascript" {
		return javascript.New(script, javascript.Options{
			Timeout: time.Second,
		})
	}
	panic("unsupported language: " + language)
}

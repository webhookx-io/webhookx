package function

import (
	"github.com/webhookx-io/webhookx/plugins/function/api"
	"github.com/webhookx-io/webhookx/plugins/function/function/javascript"
	"time"
)

type Function interface {
	Execute(ctx *api.ExecutionContext) (api.ExecutionResult, error)
}

func New(language string, script string) Function {
	if language == "javascript" {
		return javascript.New(script, javascript.Options{
			Timeout: time.Second,
		})
	}
	panic("unsupported language: " + language)
}

package sdk

import (
	"github.com/webhookx-io/webhookx/db/entities"
	"net/http"
)

type SDK struct {
	version string

	Request  *RequestSDK  `json:"request"`
	Response *ResponseSDK `json:"response"`
	Utils    *UtilsSDK    `json:"utils"`
	Log      *LogSDK      `json:"log"`

	opts *Options
}

type Options struct {
	Context *ExecutionContext
	Result  *ExecutionResult
}

func NewSDK(opts *Options) *SDK {
	return &SDK{
		version:  "0.1.0",
		Request:  NewRequestSDK(opts),
		Utils:    NewUtilsSDK(),
		Log:      NewLogSDK(),
		Response: NewResponseSDK(opts),
		opts:     opts,
	}
}

type HTTPRequest struct {
	R    *http.Request
	Body []byte
}

type HTTPResponse struct {
	Code    int
	Headers map[string]string
	Body    string
}

type ExecutionContext struct {
	HTTPRequest *HTTPRequest

	Workspace *entities.Workspace
	Source    *entities.Source
	Event     *entities.Event
}

type ExecutionResult struct {
	ReturnValue  interface{}
	HTTPResponse *HTTPResponse
}

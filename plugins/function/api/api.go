package api

import (
	"github.com/webhookx-io/webhookx/db/entities"
)

type API struct {
	version string

	Request  *RequestAPI  `json:"request"`
	Response *ResponseAPI `json:"response"`
	Utils    *UtilsAPI    `json:"utils"`
	Log      *LogAPI      `json:"log"`

	opts *Options
}

type Options struct {
	Context *ExecutionContext
	Result  *ExecutionResult
}

func NewAPI(opts *Options) *API {
	return &API{
		version:  "0.1.0",
		Request:  NewRequestAPI(opts),
		Utils:    NewUtilsAPI(),
		Log:      NewLogger(),
		Response: NewResponseAPI(opts),
		opts:     opts,
	}
}

func (api *API) GetSource() *entities.Source {
	return api.opts.Context.Source
}

func (api *API) GetEvent() *entities.Event {
	return api.opts.Context.Event
}

type HTTPRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
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

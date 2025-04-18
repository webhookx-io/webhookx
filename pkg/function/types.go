package function

import "github.com/webhookx-io/webhookx/db/entities"

type HTTPRequest struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    string
}

type HTTPResponse struct {
	Code    int
	Headers map[string]string
	Body    string
}

type ExecutionContext struct {
	HTTPRequest HTTPRequest

	Workspace *entities.Workspace
	Source    *entities.Source
	Event     *entities.Event
}

type ExecutionResult struct {
	ReturnValue  interface{}
	HTTPResponse *HTTPResponse
	Payload      *string
}

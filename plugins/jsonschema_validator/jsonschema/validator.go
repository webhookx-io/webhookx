package jsonschema

import (
	"net/http"
)

type Validator interface {
	Validate(ctx *ValidatorContext) error
}

type ValidatorContext struct {
	HTTPRequest *HTTPRequest
}

type HTTPRequest struct {
	R    *http.Request
	Data map[string]any
}

package verifier

import (
	"context"
	"net/http"
)

// Request verify request
type Request struct {
	R *http.Request
}

// Result verify result
type Result struct {
	Verified bool
	Response *Response
}

// Response verify http response
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

type Verifier interface {
	Verify(context.Context, *Request, map[string]interface{}) (*Result, error)
}

type VerifyFunc func(context.Context, *Request, map[string]interface{}) (*Result, error)

func (f VerifyFunc) Verify(ctx context.Context, req *Request, config map[string]interface{}) (*Result, error) {
	return f(ctx, req, config)
}

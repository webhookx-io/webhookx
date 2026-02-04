package deliverer

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Deliverer interface {
	Send(ctx context.Context, request *Request) (res *Response)
}

type Request struct {
	Request *http.Request
	Body    []byte
	Timeout time.Duration
}

type AclDecision struct {
	Denied bool
}

type Response struct {
	Request         *Request
	ACL             AclDecision
	StatusCode      int
	Header          http.Header
	ResponseBody    []byte
	Error           error
	Latancy         time.Duration
	ProxyStatusCode int
}

func (r *Response) Is2xx() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

func (r *Response) String() string {
	return fmt.Sprintf("%s %s %d %dms",
		r.Request.Request.Method,
		r.Request.Request.URL,
		r.StatusCode,
		r.Latancy.Milliseconds(),
	)
}

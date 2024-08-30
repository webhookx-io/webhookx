package deliverer

import (
	"fmt"
	"net/http"
	"time"
)

type Deliverer interface {
	Deliver(req *Request) (res *Response)
}

type Request struct {
	URL     string
	Method  string
	Payload []byte
	Headers map[string]string
	Timeout time.Duration
}

type Response struct {
	Request      *Request
	StatusCode   int
	Header       http.Header
	ResponseBody []byte
	Error        error
}

func (r *Response) Is2xx() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

func (r *Response) String() string {
	return fmt.Sprintf("%s %s %d", r.Request.Method, r.Request.URL, r.StatusCode)
}

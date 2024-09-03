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
	Request *http.Request
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
	Latancy      time.Duration
}

func (r *Response) Is2xx() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

func (r *Response) String() string {
	return fmt.Sprintf("%s %s %d %dms", r.Request.Method, r.Request.URL, r.StatusCode, r.Latancy.Milliseconds())
}

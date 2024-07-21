package deliverer

import (
	"fmt"
	"net/http"
)

type Deliverer interface {
	Deliver(req *Request) (res *Response, err error)
}

// Request is HTTP request
type Request struct {
	URL     string
	Method  string
	Payload []byte
	Headers map[string]string
}

// Response is HTTP response
type Response struct {
	Status       string
	StatusCode   int
	Header       http.Header
	ResponseBody []byte
	Request      *Request
	resp         *http.Response
}

func (r *Response) Is2xx() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

func (r *Response) String() string {
	return fmt.Sprintf("%s %s %d", r.Request.Method, r.Request.URL, r.StatusCode)
}

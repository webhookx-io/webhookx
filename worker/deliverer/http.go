package deliverer

import (
	"bytes"
	"github.com/webhookx-io/webhookx/config"
	"io"
	"net/http"
	"time"
)

const (
	DefaultTimeout = time.Second * 10
)

var defaultHeaders = map[string]string{
	"User-Agent":   "WebhookX/" + config.VERSION,
	"Content-Type": "application/json; charset=utf-8",
}

// HTTPDeliverer delivers via HTTP
type HTTPDeliverer struct {
	client *http.Client
}

func NewHTTPDeliverer(client *http.Client) *HTTPDeliverer {
	if client == nil {
		client = &http.Client{
			Timeout: DefaultTimeout,
		}
	}
	return &HTTPDeliverer{
		client: client,
	}
}

func (d *HTTPDeliverer) Deliver(req *Request) (res *Response, err error) {
	res = &Response{
		Request: req,
	}

	request, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer(req.Payload))
	if err != nil {
		return
	}
	for name, value := range defaultHeaders {
		request.Header.Add(name, value)
	}
	for name, value := range req.Headers {
		request.Header.Add(name, value)
	}

	resp, err := d.client.Do(request)
	if err != nil {
		return
	}
	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return
	}

	res.resp = resp
	res.Status = resp.Status
	res.StatusCode = resp.StatusCode
	res.Header = resp.Header
	res.ResponseBody = body

	return res, nil
}

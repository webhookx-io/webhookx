package deliverer

import (
	"bytes"
	"context"
	"github.com/webhookx-io/webhookx/config"
	"io"
	"net/http"
	"time"
)

var defaultHeaders = map[string]string{
	"User-Agent":   "WebhookX/" + config.VERSION,
	"Content-Type": "application/json; charset=utf-8",
}

// HTTPDeliverer delivers via HTTP
type HTTPDeliverer struct {
	defaultTimeout time.Duration
	client         *http.Client
}

func NewHTTPDeliverer(cfg *config.WorkerDeliverer) *HTTPDeliverer {
	client := &http.Client{}

	return &HTTPDeliverer{
		defaultTimeout: time.Duration(cfg.Timeout) * time.Millisecond,
		client:         client,
	}
}

func (d *HTTPDeliverer) Deliver(req *Request) (res *Response) {
	timeout := req.Timeout
	if timeout == 0 {
		timeout = d.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res = &Response{
		Request: req,
	}

	request, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewBuffer(req.Payload))
	if err != nil {
		res.Error = err
		return
	}

	req.Request = request
	for name, value := range defaultHeaders {
		request.Header.Add(name, value)
	}
	for name, value := range req.Headers {
		request.Header.Add(name, value)
	}

	response, err := d.client.Do(request)
	if err != nil {
		res.Error = err
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		res.Error = err
		return
	}
	response.Body.Close()

	res.StatusCode = response.StatusCode
	res.Header = response.Header
	res.ResponseBody = body

	return
}

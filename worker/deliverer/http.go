package deliverer

import (
	"bytes"
	"context"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/constants"
	"io"
	"net/http"
	"time"
)

// HTTPDeliverer delivers via HTTP
type HTTPDeliverer struct {
	defaultTimeout time.Duration
	client         *http.Client
}

func NewHTTPDeliverer(cfg *config.WorkerDeliverer) *HTTPDeliverer {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &HTTPDeliverer{
		defaultTimeout: time.Duration(cfg.Timeout) * time.Millisecond,
		client:         client,
	}
}

func timing(fn func()) time.Duration {
	start := time.Now()
	fn()
	stop := time.Now()
	return time.Duration(stop.UnixNano() - start.UnixNano())
}

func (d *HTTPDeliverer) Deliver(ctx context.Context, req *Request) (res *Response) {
	timeout := req.Timeout
	if timeout == 0 {
		timeout = d.defaultTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
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
	for _, header := range constants.DefaultDelivererRequestHeaders {
		request.Header.Add(header.Name, header.Value)
	}
	for name, value := range req.Headers {
		request.Header.Add(name, value)
	}

	t := timing(func() {
		response, err := d.client.Do(request)
		if err != nil {
			res.Error = err
			return
		}
		res.StatusCode = response.StatusCode
		res.Header = response.Header

		body, err := io.ReadAll(response.Body)
		if err != nil {
			res.Error = err
			return
		}
		response.Body.Close()
		res.ResponseBody = body
	})

	res.Latancy = t

	return
}

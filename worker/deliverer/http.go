package deliverer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/constants"
	"io"
	"net"
	"net/http"
	"net/netip"
	"time"
)

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var DefaultResolver Resolver = net.DefaultResolver

type contextKey struct{}

// HTTPDeliverer delivers via HTTP
type HTTPDeliverer struct {
	defaultTimeout time.Duration
	client         *http.Client
}

func restrictedDialFunc(acl *ACL) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		ips, err := DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			if acl.Allow(host, ip) {
				return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			}
		}

		if res, ok := ctx.Value(contextKey{}).(*Response); ok {
			res.ACL.Denied = true
		}

		return nil, fmt.Errorf("request to %s(ip=%s) is denied", addr, ips[0])
	}
}

func NewHTTPDeliverer(cfg *config.WorkerDeliverer) *HTTPDeliverer {
	transport := &http.Transport{
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext:           restrictedDialFunc(NewACL(AclOptions{Rules: cfg.ACL.Deny})),
	}
	client := &http.Client{
		Transport: transport,
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

	ctx = context.WithValue(ctx, contextKey{}, res)
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

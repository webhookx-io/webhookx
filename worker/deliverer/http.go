package deliverer

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/webhookx-io/webhookx/constants"
	"go.uber.org/zap"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"time"
)

type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

var DefaultResolver Resolver = net.DefaultResolver
var DefaultTLSConfig *tls.Config = nil

type contextKey struct{}

// HTTPDeliverer delivers via HTTP
type HTTPDeliverer struct {
	log            *zap.SugaredLogger
	requestTimeout time.Duration
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

type ProxyOptions struct {
	URL              string
	TLSCert          string
	TLSKey           string
	TLSCaCertificate string
	TLSVerify        bool
}

type AccessControlOptions struct {
	Deny []string
}

type Options struct {
	Logger               *zap.SugaredLogger
	RequestTimeout       time.Duration
	AccessControlOptions AccessControlOptions
}

func NewHTTPDeliverer(opts Options) *HTTPDeliverer {
	transport := &http.Transport{
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext:           restrictedDialFunc(NewACL(AclOptions{Rules: opts.AccessControlOptions.Deny})),
		TLSClientConfig:       DefaultTLSConfig,
	}
	client := &http.Client{
		Transport: transport,
	}

	return &HTTPDeliverer{
		log:            opts.Logger,
		requestTimeout: opts.RequestTimeout,
		client:         client,
	}
}

func (d *HTTPDeliverer) SetupProxy(opts ProxyOptions) error {
	proxyURL, err := url.Parse(opts.URL)
	if err != nil {
		return fmt.Errorf("invalid proxy url '%s': %s", opts.URL, err)
	}

	transport := d.client.Transport.(*http.Transport)

	transport.Proxy = http.ProxyURL(proxyURL)
	transport.DialContext = nil
	transport.OnProxyConnectResponse = func(ctx context.Context, proxyURL *url.URL, connectReq *http.Request, connectRes *http.Response) error {
		if connectRes.StatusCode != 200 {
			if res, ok := ctx.Value(contextKey{}).(*Response); ok {
				res.ProxyStatusCode = connectRes.StatusCode
			}
		}
		return nil
	}

	if proxyURL.Scheme == "https" {
		tlsConfig := &tls.Config{
			ServerName:         proxyURL.Hostname(),
			InsecureSkipVerify: opts.TLSVerify,
		}
		if opts.TLSCert != "" || opts.TLSKey != "" {
			cert, err := tls.LoadX509KeyPair(opts.TLSCert, opts.TLSKey)
			if err != nil {
				return fmt.Errorf("failed to load client certificate: %s", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		if opts.TLSCaCertificate != "" {
			caPEM, err := os.ReadFile(opts.TLSCaCertificate)
			if err != nil {
				return fmt.Errorf("failed to read ca certificate: %s", err)
			}
			cp := x509.NewCertPool()
			if !cp.AppendCertsFromPEM(caPEM) {
				return fmt.Errorf("failed to append ca certificate to pool")
			}
			tlsConfig.RootCAs = cp
		}
		transport.DialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{}
			conn, err := dialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(conn, tlsConfig)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				_ = conn.Close()
				return nil, err
			}
			return tlsConn, nil
		}
	}

	d.log.Infow("proxy enabled", "proxy_url", opts.URL)

	return nil
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
		timeout = d.requestTimeout
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

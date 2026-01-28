package plugin

import (
	"context"
	"net/http"

	"github.com/webhookx-io/webhookx/pkg/http/response"
)

type Plugin interface {
	// Name returns plugin's name
	Name() string

	// Priority returns plugin's priority
	Priority() int

	// Init inits plugin with configuration
	Init(config map[string]interface{}) error

	// GetConfig returns plugin's configuration
	GetConfig() map[string]interface{}

	// ValidateConfig validates plugin's configuration
	ValidateConfig(config map[string]interface{}) error

	// ExecuteInbound executes inbound
	ExecuteInbound(c *Context) error

	// ExecuteOutbound executes outbound
	ExecuteOutbound(c *Context) error
}

func New(name string) (Plugin, bool) {
	r := GetRegistration(name)
	if r == nil {
		return nil, false
	}
	return r.Factory(), true
}

type Context struct {
	Request    *http.Request
	ctx        context.Context
	rw         http.ResponseWriter
	body       []byte
	terminated bool
}

func NewContext(ctx context.Context, r *http.Request, w http.ResponseWriter) *Context {
	return &Context{
		Request: r,
		ctx:     ctx,
		rw:      w,
	}
}

func (c *Context) GetRequestBody() []byte {
	return c.body
}

func (c *Context) SetRequestBody(body []byte) {
	c.body = body
}

func (c *Context) Response(headers map[string]string, code int, body []byte) {
	response.Response(c.rw, headers, code, body)
	c.terminated = true
}

func (c *Context) JSON(code int, obj interface{}) {
	response.JSON(c.rw, code, obj)
	c.terminated = true
}

func (c *Context) Context() context.Context {
	return c.ctx
}

func (c *Context) WithContext(ctx context.Context) *Context {
	c.ctx = ctx
	return c
}

func (c *Context) IsTerminated() bool {
	return c.terminated
}

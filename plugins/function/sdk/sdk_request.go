package sdk

import (
	"github.com/dop251/goja"
	"github.com/webhookx-io/webhookx/utils"
)

type RequestSDK struct {
	opts *Options
}

func NewRequestSDK(opts *Options) *RequestSDK {
	return &RequestSDK{
		opts: opts,
	}
}

func (sdk *RequestSDK) GetHost() string {
	return sdk.opts.Context.HTTPRequest.R.Host
}

func (sdk *RequestSDK) GetMethod() string {
	return sdk.opts.Context.HTTPRequest.R.Method
}

func (sdk *RequestSDK) GetPath() string {
	return sdk.opts.Context.HTTPRequest.R.URL.Path
}

func (sdk *RequestSDK) GetHeaders() map[string]string {
	return utils.HeaderMap(sdk.opts.Context.HTTPRequest.R.Header)
}

func (sdk *RequestSDK) getHeader(name string) *string {
	values := sdk.opts.Context.HTTPRequest.R.Header.Values(name)
	if len(values) == 0 {
		return nil
	}
	value := values[0]
	return &value
}

func (sdk *RequestSDK) GetHeader(call goja.FunctionCall) goja.Value {
	name := call.Argument(0).String()
	value := sdk.getHeader(name)
	if value == nil {
		return goja.Null()
	}
	return sdk.opts.VM.ToValue(*value)
}

func (sdk *RequestSDK) GetBody() string {
	return string(sdk.opts.Context.HTTPRequest.Body)
}

func (sdk *RequestSDK) SetBody(body string) {
	sdk.opts.Context.HTTPRequest.Body = []byte(body)
}

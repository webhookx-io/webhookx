package sdk

import "github.com/webhookx-io/webhookx/utils"

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

func (sdk *RequestSDK) GetHeader(name string) string {
	return sdk.opts.Context.HTTPRequest.R.Header.Get(name)
}

func (sdk *RequestSDK) GetBody() string {
	return string(sdk.opts.Context.HTTPRequest.Body)
}

func (sdk *RequestSDK) SetBody(body string) {
	sdk.opts.Context.HTTPRequest.Body = []byte(body)
}

package sdk

type RequestSDK struct {
	opts *Options
}

func NewRequestSDK(opts *Options) *RequestSDK {
	return &RequestSDK{
		opts: opts,
	}
}

func (sdk *RequestSDK) GetMethod() string {
	return sdk.opts.Context.HTTPRequest.Method
}

func (sdk *RequestSDK) GetHeaders() map[string]string {
	return sdk.opts.Context.HTTPRequest.Headers
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

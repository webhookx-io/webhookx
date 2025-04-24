package api

type RequestAPI struct {
	opts *Options
}

func NewRequestAPI(opts *Options) *RequestAPI {
	return &RequestAPI{
		opts: opts,
	}
}

func (api *RequestAPI) GetMethod() string {
	return api.opts.Context.HTTPRequest.Method
}

func (api *RequestAPI) GetHeaders() map[string]string {
	return api.opts.Context.HTTPRequest.Headers
}

func (api *RequestAPI) GetHeader(name string) string {
	return api.opts.Context.HTTPRequest.Headers[name]
}

func (api *RequestAPI) GetBody() string {
	return string(api.opts.Context.HTTPRequest.Body)
}

func (api *RequestAPI) SetBody(body string) {
	api.opts.Context.HTTPRequest.Body = []byte(body)
}

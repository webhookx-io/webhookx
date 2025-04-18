package function

type RequestAPI struct {
	ctx *ExecutionContext
}

func NewRequestAPI(ctx *ExecutionContext, _ *ExecutionResult) *RequestAPI {
	return &RequestAPI{
		ctx: ctx,
	}
}

func (api *RequestAPI) GetMethod() string {
	return api.ctx.HTTPRequest.Method
}

func (api *RequestAPI) GetBody() string {
	return api.ctx.HTTPRequest.Body
}

func (api *RequestAPI) GetHeaders() map[string]string {
	return api.ctx.HTTPRequest.Headers
}

func (api *RequestAPI) GetHeader(name string) string {
	return api.ctx.HTTPRequest.Headers[name]
}

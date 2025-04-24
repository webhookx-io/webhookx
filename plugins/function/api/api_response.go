package api

import "encoding/json"

type ResponseAPI struct {
	opts *Options
}

func NewResponseAPI(opts *Options) *ResponseAPI {
	return &ResponseAPI{
		opts: opts,
	}
}

func (api *ResponseAPI) Exit(code int, headers map[string]string, body interface{}) {
	response := &HTTPResponse{
		Code:    code,
		Headers: headers,
	}
	switch v := body.(type) {
	case string:
		response.Body = v
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		response.Body = string(bytes)
	}
	api.opts.Result.HTTPResponse = response
}

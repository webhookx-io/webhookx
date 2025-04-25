package sdk

import "encoding/json"

type ResponseSDK struct {
	opts *Options
}

func NewResponseSDK(opts *Options) *ResponseSDK {
	return &ResponseSDK{
		opts: opts,
	}
}

func (sdk *ResponseSDK) Exit(code int, headers map[string]string, body interface{}) {
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
	sdk.opts.Result.HTTPResponse = response
}

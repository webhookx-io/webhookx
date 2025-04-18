package function

import "encoding/json"

type ResponseAPI struct {
	res *ExecutionResult
}

func NewResponseAPI(_ *ExecutionContext, res *ExecutionResult) *ResponseAPI {
	return &ResponseAPI{
		res: res,
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
	api.res.HTTPResponse = response
}

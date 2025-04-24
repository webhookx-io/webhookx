package function

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/function/api"
	"github.com/webhookx-io/webhookx/plugins/function/function"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
)

type Config struct {
	Function string `json:"function" validate:"required"`
}

type FunctionPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &FunctionPlugin{}
	p.Name = "function"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *FunctionPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *FunctionPlugin) ExecuteOutbound(req *plugin.OutboundRequest, _ *plugin.Context) error {
	panic("not implemented")
}

func (p *FunctionPlugin) ExecuteInbound(r *http.Request, body []byte, w http.ResponseWriter) (result plugin.InboundResult, err error) {
	fn := function.New("javascript", p.Config.Function)
	result.Payload = body

	req := api.HTTPRequest{
		R:       r,
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: utils.HeaderMap(r.Header),
		Body:    body,
	}

	res, err := fn.Execute(&api.ExecutionContext{
		HTTPRequest: &req,
		Workspace:   nil,
		Source:      nil,
		Event:       nil,
	})
	if err != nil {
		return
	}

	if res.HTTPResponse != nil {
		for k, v := range res.HTTPResponse.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(res.HTTPResponse.Code)
		_, _ = w.Write([]byte(res.HTTPResponse.Body))
		result.Terminated = true
		return
	}

	result.Payload = req.Body
	return
}

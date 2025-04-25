package function

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/function/function"
	"github.com/webhookx-io/webhookx/plugins/function/sdk"
	"github.com/webhookx-io/webhookx/utils"
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

func (p *FunctionPlugin) ExecuteInbound(inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
	fn := function.New("javascript", p.Config.Function)

	req := sdk.HTTPRequest{
		R:    inbound.Request,
		Body: inbound.RawBody,
	}

	res, err := fn.Execute(&sdk.ExecutionContext{
		HTTPRequest: &req,
	})
	if err != nil {
		return
	}

	if res.HTTPResponse != nil {
		for k, v := range res.HTTPResponse.Headers {
			inbound.Response.Header().Set(k, v)
		}
		inbound.Response.WriteHeader(res.HTTPResponse.Code)
		_, _ = inbound.Response.Write([]byte(res.HTTPResponse.Body))
		result.Terminated = true
		return
	}

	result.Payload = req.Body
	return
}

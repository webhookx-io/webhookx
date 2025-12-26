package function

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/function/function"
	"github.com/webhookx-io/webhookx/plugins/function/sdk"
)

type Config struct {
	Function string `json:"function"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("FunctionPluginConfiguration")
}

type FunctionPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *FunctionPlugin) Name() string {
	return "function"
}

func (p *FunctionPlugin) Priority() int {
	return 80
}

func (p *FunctionPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
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

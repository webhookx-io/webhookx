package function

import (
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

func (p *FunctionPlugin) ExecuteInbound(c *plugin.Context) error {
	fn := function.New("javascript", p.Config.Function)

	req := sdk.HTTPRequest{
		R:    c.Request,
		Body: c.GetRequestBody(),
	}
	res, err := fn.Execute(&sdk.ExecutionContext{
		HTTPRequest: &req,
	})
	if err != nil {
		return err
	}

	c.SetRequestBody(req.Body)
	if res.HTTPResponse != nil {
		c.Response(res.HTTPResponse.Headers, res.HTTPResponse.Code, []byte(res.HTTPResponse.Body))
	}

	return nil
}

package basic_auth

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Config struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("BasicAuthPluginConfiguration")
}

type BasicAuthPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *BasicAuthPlugin) Name() string {
	return "basic-auth"
}

func (p *BasicAuthPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
	username, password, ok := inbound.Request.BasicAuth()
	if !ok || username != p.Config.Username || password != p.Config.Password {
		response.JSON(inbound.Response, 401, `{"message":"Unauthorized"}`)
		result.Terminated = true
	}

	result.Payload = inbound.RawBody
	return
}

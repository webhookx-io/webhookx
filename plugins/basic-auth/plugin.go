package basic_auth

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
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

func (p *BasicAuthPlugin) Priority() int {
	return 109
}

func (p *BasicAuthPlugin) ExecuteInbound(c *plugin.Context) error {
	username, password, ok := c.Request.BasicAuth()
	if !ok || username != p.Config.Username || password != p.Config.Password {
		c.JSON(401, `{"message":"Unauthorized"}`)
	}
	return nil
}

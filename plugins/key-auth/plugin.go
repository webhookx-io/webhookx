package key_auth

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Config struct {
	ParamName      string   `json:"param_name"`
	ParamLocations []string `json:"param_locations"`
	Key            string   `json:"key"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("KeyAuthPluginConfiguration")
}

type KeyAuthPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *KeyAuthPlugin) Name() string {
	return "key-auth"
}

func (p *KeyAuthPlugin) Priority() int {
	return 108
}

func (p *KeyAuthPlugin) ExecuteInbound(c *plugin.Context) error {
	name := p.Config.ParamName
	key := p.Config.Key

	querys := c.Request.URL.Query()
	headers := c.Request.Header

	found := false
	for _, source := range p.Config.ParamLocations {
		var value string
		switch source {
		case "query":
			value = querys.Get(name)
		case "header":
			value = headers.Get(name)
		}
		if value == key {
			found = true
			break
		}
	}

	if !found {
		c.JSON(401, `{"message":"Unauthorized"}`)
	}

	return nil
}

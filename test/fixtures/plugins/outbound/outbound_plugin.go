package outbound

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Config struct {
}

func (c Config) Schema() *openapi3.Schema {
	return openapi3.NewObjectSchema()
}

type OutboundPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *OutboundPlugin) Name() string {
	return "outbound"
}

func (p *OutboundPlugin) Priority() int {
	return 0
}

func (p *OutboundPlugin) ExecuteOutbound(c *plugin.Context) error {
	return nil
}

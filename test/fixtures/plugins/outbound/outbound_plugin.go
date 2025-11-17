package outbound

import (
	"context"
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

func (p *OutboundPlugin) ExecuteOutbound(ctx context.Context, outbound *plugin.Outbound) error {
	return nil
}

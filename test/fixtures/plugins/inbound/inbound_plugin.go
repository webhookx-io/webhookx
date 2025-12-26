package inbound

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

type InboundPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *InboundPlugin) Name() string {
	return "inbound"
}

func (p *InboundPlugin) Priority() int {
	return 0
}

func (p *InboundPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	return
}

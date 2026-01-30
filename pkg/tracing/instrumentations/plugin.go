package instrumentations

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/tracing"
)

var _ plugin.Plugin = &InstrumentedPlugin{}

type InstrumentedPlugin struct {
	plugin.Plugin
}

func NewInstrumentedPlugin(plugin plugin.Plugin) plugin.Plugin {
	return &InstrumentedPlugin{Plugin: plugin}
}

func (p *InstrumentedPlugin) ExecuteInbound(c *plugin.Context) error {
	ctx, span := tracing.Start(c.Context(), "plugin."+p.Name()+".inbound")
	defer span.End()
	return p.Plugin.ExecuteInbound(c.WithContext(ctx))
}

func (p *InstrumentedPlugin) ExecuteOutbound(c *plugin.Context) error {
	ctx, span := tracing.Start(c.Context(), "plugin."+p.Name()+".outbound")
	defer span.End()
	return p.Plugin.ExecuteOutbound(c.WithContext(ctx))
}

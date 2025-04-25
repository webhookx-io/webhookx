package inbound

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
)

type Config struct {
}

type InboundPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &InboundPlugin{}
	p.Name = "inbound"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *InboundPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *InboundPlugin) ExecuteInbound(inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	return
}

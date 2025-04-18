package inbound

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
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

func (p *InboundPlugin) ExecuteOutbound(req *plugin.Request, _ *plugin.Context) error {
	panic("not implemented")
}

func (p *InboundPlugin) ExecuteInbound(r *http.Request, w http.ResponseWriter) error {
	return nil
}

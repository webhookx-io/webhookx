package outbound

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
)

type Config struct {
}

type OutboundPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &OutboundPlugin{}
	p.Name = "outbound"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *OutboundPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *OutboundPlugin) ExecuteOutbound(req *plugin.Request, _ *plugin.Context) error {
	return nil
}

func (p *OutboundPlugin) ExecuteInbound(r *http.Request, w http.ResponseWriter) error {
	panic("not implemented")
}

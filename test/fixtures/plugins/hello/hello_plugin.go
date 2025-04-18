package hello

import (
	"fmt"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
)

type Config struct {
	Message string `json:"message" validate:"required"`
}

type HelloPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &HelloPlugin{}
	p.Name = "hello"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *HelloPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *HelloPlugin) ExecuteOutbound(req *plugin.Request, _ *plugin.Context) error {
	fmt.Println(p.Config.Message)
	return nil
}

func (p *HelloPlugin) ExecuteInbound(r *http.Request, w http.ResponseWriter) error {
	fmt.Println(p.Config.Message)
	return nil
}

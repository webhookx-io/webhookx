package function

import (
	"bytes"
	"github.com/webhookx-io/webhookx/pkg/function"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"io"
	"net/http"
)

type Config struct {
	Function string `json:"function" validate:"required"`
}

type FunctionPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &FunctionPlugin{}
	p.Name = "function"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *FunctionPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *FunctionPlugin) ExecuteOutbound(req *plugin.Request, _ *plugin.Context) error {
	panic("not implemented")
}

func (p *FunctionPlugin) ExecuteInbound(r *http.Request, w http.ResponseWriter) error {
	fn := function.New("javascript", p.Config.Function)
	res, err := fn.Execute(nil) // todo
	if err != nil {
		return err
	}
	if res.HTTPResponse != nil {
		for k, v := range res.HTTPResponse.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(res.HTTPResponse.Code)
		w.Write([]byte(res.HTTPResponse.Body))
		return nil
	}
	if res.Payload != nil {
		r.Body = io.NopCloser(bytes.NewBufferString("new body"))
	}
	return nil
}

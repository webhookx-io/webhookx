package mock

import (
	"context"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

var schemaJSON = `
{
    "type": "object",
    "properties": {
		"status": {
			"type": "integer"
		},
		"headers": {
			"type": "object",
			"additionalProperties": {
				"type": "string"
			}
        },
		"body": {
            "type": "string"
        }
    }
}
`

type Config struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (c Config) Schema() *openapi3.Schema {
	schema := openapi3.NewSchema()
	err := schema.UnmarshalJSON([]byte(schemaJSON))
	if err != nil {
		panic(err)
	}
	return schema
}

type Plugin struct {
	plugin.BasePlugin[Config]
}

func (p *Plugin) Name() string {
	return "mock"
}

func (p *Plugin) Priority() int {
	return 0
}

func (p *Plugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	for k, v := range p.Config.Headers {
		inbound.Response.Header().Set(k, v)
	}
	inbound.Response.WriteHeader(p.Config.Status)
	_, _ = inbound.Response.Write([]byte(p.Config.Body))
	res.Terminated = true

	return
}

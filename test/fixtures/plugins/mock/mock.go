package mock

import (
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

func (p *Plugin) ExecuteInbound(c *plugin.Context) error {
	c.Response(p.Config.Headers, p.Config.Status, []byte(p.Config.Body))
	return nil
}

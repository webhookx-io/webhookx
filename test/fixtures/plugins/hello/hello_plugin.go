package hello

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

var schemaJSON = `
{
    "type": "object",
    "properties": {
        "message": {
            "type": "string"
        }
    },
   "required": ["message"]
}
`

type Config struct {
	Message string `json:"message"`
}

func (c Config) Schema() *openapi3.Schema {
	schema := openapi3.NewSchema()
	err := schema.UnmarshalJSON([]byte(schemaJSON))
	if err != nil {
		panic(err)
	}
	return schema
}

type HelloPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *HelloPlugin) Name() string {
	return "hello"
}

func (p *HelloPlugin) Priority() int {
	return 0
}

func (p *HelloPlugin) ExecuteOutbound(c *plugin.Context) error {
	fmt.Println(p.Config.Message)
	return nil
}

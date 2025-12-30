package event_validation

import (
	"context"
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins/event-validation/validator"
)

type Config struct {
	Version         string             `json:"version"`
	VerboseResponse bool               `json:"verbose_response"`
	Schemas         map[string]*string `json:"schemas"`
	DefaultSchema   *string            `json:"default_schema"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("EventValidationPluginConfiguration")
}

type EventValidationPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *EventValidationPlugin) Name() string {
	return "event-validation"
}

func (p *EventValidationPlugin) Priority() int {
	return 90
}

func (p *EventValidationPlugin) ValidateConfig(config map[string]interface{}) error {
	return p.BasePlugin.ValidateConfig(config)
}

func (p *EventValidationPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	var event map[string]any
	body := inbound.RawBody
	if err = json.Unmarshal(body, &event); err != nil {
		return
	}

	eventType, _ := event["event_type"].(string)
	data := event["data"]

	schema := p.Config.Schemas[eventType]
	if schema == nil {
		schema = p.Config.DefaultSchema
	}

	if schema == nil {
		// no schema found
		res.Payload = body
		return
	}

	v, err := validator.NewValidator(p.Config.Version, *schema)
	if err != nil {
		return
	}
	validateErr := v.Validate(data)
	if validateErr != nil {
		resp := &types.ErrorResponse{
			Message: "event data does not conform to schema",
		}
		if p.Config.VerboseResponse {
			if e, ok := validateErr.(validator.ValidateError); ok {
				resp.Error = e.Errors()
			} else {
				resp.Error = validateErr.Error()
			}
		}
		response.JSON(inbound.Response, 400, resp)
		res.Terminated = true
		return
	}

	res.Payload = body
	return
}

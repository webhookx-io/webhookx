package jsonschema_validator

import (
	"context"
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator/jsonschema"
)

type Config struct {
	Draft         string             `json:"draft"`
	DefaultSchema string             `json:"default_schema"`
	Schemas       map[string]*Schema `json:"schemas"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("JsonschemaValidatorPluginConfiguration")
}

type Schema struct {
	Schema string `json:"schema"`
}

type SchemaValidatorPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *SchemaValidatorPlugin) Name() string {
	return "jsonschema-validator"
}

func (p *SchemaValidatorPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	var event map[string]any
	body := inbound.RawBody
	if err = json.Unmarshal(body, &event); err != nil {
		return
	}

	eventType, ok := event["event_type"].(string)
	if !ok || eventType == "" {
		res.Payload = body
		return
	}

	data := event["data"]
	if data == nil {
		res.Payload = body
		return
	}

	schema, ok := p.Config.Schemas[eventType]
	if !ok {
		res.Payload = body
		return
	}
	if schema == nil || schema.Schema == "" {
		if p.Config.DefaultSchema == "" {
			res.Payload = body
			return
		}
		schema = &Schema{
			Schema: p.Config.DefaultSchema,
		}
	}

	validator := jsonschema.New([]byte(schema.Schema))
	e := validator.Validate(&jsonschema.ValidatorContext{
		HTTPRequest: &jsonschema.HTTPRequest{
			R:    inbound.Request,
			Data: data.(map[string]any),
		},
	})
	if e != nil {
		response.JSON(inbound.Response, 400, types.ErrorResponse{
			Message: "Request Validation",
			Error:   e,
		})
		res.Terminated = true
		return
	}
	res.Payload = body
	return
}

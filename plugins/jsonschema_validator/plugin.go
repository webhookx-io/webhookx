package jsonschema_validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator/jsonschema"
	"github.com/webhookx-io/webhookx/utils"
	"io"
	"net/http"
	"os"
	"time"
)

type Config struct {
	Schemas map[string]*SchemaResource `json:"schemas" validate:"dive,required"`
}

type EventTypeSchema struct {
	EventType  string `json:"event_type" validate:"required,max=100"`
	JSONSchema string `json:"jsonschema" validate:"required,jsonschema,max=1048576"`
}

type SchemaResource struct {
	JSONString string `json:"json" validate:"omitempty,json,max=1048576"`
	File       string `json:"file" validate:"omitempty,file"`
	URL        string `json:"url" validate:"omitempty,url"`
}

var cache, _ = lru.New[string, []byte](128)

func (s *SchemaResource) Resource() ([]byte, string, error) {
	// priority: json > file > url
	if s.JSONString != "" {
		return []byte(s.JSONString), "json", nil
	}
	if s.File != "" {
		bytes, ok := cache.Get(s.File)
		if ok {
			return bytes, "file", nil
		}
		bytes, err := os.ReadFile(s.File)
		if err != nil {
			return nil, "file", fmt.Errorf("failed to read schema: %w", err)
		}
		cache.Add(s.File, bytes)
		return bytes, "file", nil
	}
	if s.URL != "" {
		bytes, ok := cache.Get(s.URL)
		if ok {
			return bytes, "url", nil
		}
		client := &http.Client{
			Timeout: time.Second * 2,
		}
		resp, err := client.Get(s.URL)
		if err != nil {
			return nil, "url", fmt.Errorf("failed to fetch schema: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, "url", fmt.Errorf("failed to read schema from response: %w", err)
		}
		cache.Add(s.URL, body)
		return body, "url", nil
	}
	return nil, "json", errors.New("no schema defined")
}

type SchemaValidatorPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &SchemaValidatorPlugin{}
	p.Name = "jsonschema-validator"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *SchemaValidatorPlugin) ValidateConfig() error {
	err := utils.Validate(p.Config)
	if err != nil {
		return err
	}

	e := errs.NewValidateError(errors.New("request validation"))
	for event, schema := range p.Config.Schemas {
		field := fmt.Sprintf("schemas[%s]", event)
		if schema == nil {
			e.Fields[field] = fmt.Errorf("schema is empty")
			return e
		}
		schemaBytes, invalidField, err := schema.Resource()
		if err != nil {
			e.Fields[field] = map[string]string{
				invalidField: err.Error(),
			}
			return e
		}
		openapiSchema := &openapi3.Schema{}
		err = openapiSchema.UnmarshalJSON(schemaBytes)
		if err != nil {
			e.Fields[field] = map[string]string{
				invalidField: "the content must be a valid json string",
			}
			return e
		}
		err = openapiSchema.Validate(context.Background(), openapi3.EnableSchemaFormatValidation())
		if err != nil {
			e.Fields[field] = map[string]string{
				invalidField: fmt.Sprintf("invalid jsonschema: %v", err),
			}
			return e
		}
	}
	return nil
}

func (p *SchemaValidatorPlugin) ExecuteInbound(inbound *plugin.Inbound) (res plugin.InboundResult, err error) {
	// parse body to get event type
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

	schemaResource, ok := p.Config.Schemas[eventType]
	if !ok || schemaResource == nil {
		res.Payload = body
		return
	}

	bytes, _, err := schemaResource.Resource()
	if err != nil {
		return
	}
	validator := jsonschema.New(bytes)
	err = validator.Validate(&jsonschema.ValidatorContext{
		HTTPRequest: &jsonschema.HTTPRequest{
			R:    inbound.Request,
			Data: data.(map[string]any),
		},
	})
	if err != nil {
		return
	}
	res.Payload = body
	return
}

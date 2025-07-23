package entities

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/openapi"
)

var schemaNames = []string{"Endpoint", "Source", "Workspace"}
var schemas map[string]*JSONSchema

type JSONSchema struct {
	defaultJSON string
	schema      *openapi3.Schema
}

func (s *JSONSchema) Defaults() map[string]interface{} {
	defaults := make(map[string]interface{})
	err := json.Unmarshal([]byte(s.defaultJSON), &defaults)
	if err != nil {
		panic(err)
	}
	return defaults
}

func (s *JSONSchema) Validate(value map[string]interface{}) error {
	return openapi.Validate(s.schema, value)
}

func RegisterSchema(name string, schema *openapi3.Schema) error {
	defaults := make(map[string]interface{})
	err := openapi.SetDefaults(schema, defaults)
	if err != nil {
		return err
	}

	defaultJSON, err := json.Marshal(defaults)
	if err != nil {
		return err
	}

	if schemas == nil {
		schemas = make(map[string]*JSONSchema)
	}
	schemas[name] = &JSONSchema{
		schema:      schema,
		defaultJSON: string(defaultJSON),
	}
	return nil
}

func LookSchema(name string) *JSONSchema {
	return schemas[name]
}

func LoadOpenAPI(bytes []byte) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(bytes)
	if err != nil {
		panic(fmt.Errorf("failed to load OpenAPI document: %w", err))
	}

	if err = doc.Validate(loader.Context,
		openapi3.EnableSchemaFormatValidation(),
		openapi3.DisableSchemaDefaultsValidation(),
	); err != nil {
		panic(fmt.Errorf("OpenAPI document validation failed: %w", err))
	}

	for _, name := range schemaNames {
		schema, err := doc.Components.Schemas.JSONLookup(name)
		if err != nil {
			panic(fmt.Errorf("failed to lookup JSON schema %q: %w", name, err))
		}
		err = RegisterSchema(name, schema.(*openapi3.Schema))
		if err != nil {
			panic(fmt.Errorf("failed to register JSON schema %q: %w", name, err))
		}
	}
}

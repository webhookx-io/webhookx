package entities

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
)

var schemaNames = []string{"Endpoint", "Source", "Workspace", "Plugin", "Configuration"}
var schemas map[string]*openapi3.Schema

func RegisterSchema(name string, schema *openapi3.Schema) error {
	if schemas == nil {
		schemas = make(map[string]*openapi3.Schema)
	}
	schemas[name] = schema
	return nil
}

func LookupSchema(name string) *openapi3.Schema {
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

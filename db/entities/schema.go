package entities

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

type Schema interface {
	SchemaName() string
}

var spec *openapi3.T

func LookupSchema(name string) *openapi3.Schema {
	s, err := spec.Components.Schemas.JSONLookup(name)
	if err != nil {
		panic(fmt.Errorf("failed to lookup JSON schema %q: %w", name, err))
	}
	return s.(*openapi3.Schema)
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

	spec = doc
}

package entities

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/openapi"
)

type Schema interface {
	SchemaName() string
}

func LookupSchema(name string) *openapi3.Schema {
	s, err := openapi.Spec.Components.Schemas.JSONLookup(name)
	if err != nil {
		panic(fmt.Errorf("failed to lookup JSON schema %q: %w", name, err))
	}
	return s.(*openapi3.Schema)
}

package entities

import (
	"github.com/getkin/kin-openapi/openapi3"
)

var schemaRegistry = map[string]*openapi3.Schema{}

func RegisterSchema(name string, schema *openapi3.Schema) {
	schemaRegistry[name] = schema
}

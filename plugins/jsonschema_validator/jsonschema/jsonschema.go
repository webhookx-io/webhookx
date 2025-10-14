package jsonschema

import (
	"github.com/getkin/kin-openapi/openapi3"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/utils"
)

type JSONSchema struct {
	schemaDef string
	hex       string
}

func New(schemaDef []byte) *JSONSchema {
	return &JSONSchema{
		schemaDef: string(schemaDef),
		hex:       utils.Hash256(string(schemaDef)),
	}
}

var cache, _ = lru.New[string, *openapi3.Schema](128)

func (s *JSONSchema) Validate(ctx *ValidatorContext) error {
	schema, ok := cache.Get(s.hex)
	if !ok {
		schema = &openapi3.Schema{}
		err := schema.UnmarshalJSON([]byte(s.schemaDef))
		if err != nil {
			return err
		}
		cache.Add(s.hex, schema)
	}

	err := openapi.Validate(schema, ctx.HTTPRequest.Data)
	if err != nil {
		return err
	}
	return nil
}

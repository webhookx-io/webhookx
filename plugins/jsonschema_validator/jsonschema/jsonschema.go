package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/utils"
	"strings"
)

const (
	Draft_4     string = "draft-04"
	Draft_6     string = "draft-06"
	Draft_7     string = "draft-07"
	Draft_2019  string = "draft/2019-09"
	Draft_2020  string = "draft/2020-12"
	OpenAPI_3_0 string = "openapi-3.0"
)

type JSONSchema struct {
	hash      uint64
	version   string
	schemaDef string
	c         *jsonschema.Compiler
}

func New(schemaDef string, c *jsonschema.Compiler) *JSONSchema {
	version := getVersion(schemaDef)
	return &JSONSchema{
		hash:      utils.XXHash3(schemaDef),
		version:   version,
		schemaDef: schemaDef,
		c:         c,
	}
}

var cache, _ = lru.New[uint64, any](128)

func (s *JSONSchema) Validate(ctx *ValidatorContext) error {
	switch s.version {
	case OpenAPI_3_0:
		openapi3Schema := &openapi3.Schema{}
		if schema, ok := cache.Get(s.hash); !ok {
			err := openapi3Schema.UnmarshalJSON([]byte(s.schemaDef))
			if err != nil {
				return err
			}
			cache.Add(s.hash, openapi3Schema)
		} else {
			openapi3Schema = schema.(*openapi3.Schema)
		}

		err := openapi.Validate(openapi3Schema, ctx.HTTPRequest.Data.(map[string]interface{}))
		if err != nil {
			return err
		}
	default:
		schema, ok := cache.Get(s.hash)
		if !ok {
			doc, err := jsonschema.UnmarshalJSON(strings.NewReader(s.schemaDef))
			if err != nil {
				return err
			}
			schema = doc
			cache.Add(s.hash, schema)
		}
		resourceFile := fmt.Sprintf("%x.json", s.hash)
		err := s.c.AddResource(resourceFile, schema)
		var existErr *jsonschema.ResourceExistsError
		if err != nil && !errors.As(err, &existErr) {
			return err
		}
		sch, err := s.c.Compile(resourceFile)
		if err != nil {
			return err
		}

		err = sch.Validate(ctx.HTTPRequest.Data)
		if err != nil {
			validateErr := errs.NewValidateError(errs.ErrRequestValidation)
			validateErr.Fields = errs.ParseJSONSchemaValidationError(err.(*jsonschema.ValidationError), false)
			return validateErr
		}
	}
	return nil
}

func getVersion(schemaDef string) string {
	schemaMap := make(map[string]any)
	err := json.Unmarshal([]byte(schemaDef), &schemaMap)
	if err != nil {
		panic(err)
	}
	var schemaStr string
	if schema, exist := schemaMap["$schema"]; exist {
		if s, ok := schema.(string); ok {
			schemaStr = s
		}
	}

	switch {
	case strings.Contains(schemaStr, Draft_4):
		return Draft_4
	case strings.Contains(schemaStr, Draft_6):
		return Draft_6
	case strings.Contains(schemaStr, Draft_7):
		return Draft_7
	case strings.Contains(schemaStr, Draft_2019):
		return Draft_2019
	case strings.Contains(schemaStr, Draft_2020):
		return Draft_2020
	default:
		return OpenAPI_3_0
	}
}

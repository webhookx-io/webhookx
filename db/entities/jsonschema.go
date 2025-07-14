package entities

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/utils"
	"io"
	"reflect"
)

var schemas openapi3.Schemas

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
	schemas = doc.Components.Schemas
}

func getStructName(i interface{}) string {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t.Name()
	}
	return ""
}

// NewEntity new a default entity by OpenAPI schema definition
func NewEntity[T any](defaultSet func(*T)) T {
	var entity T
	defaultObject := make(map[string]interface{})
	val, err := schemas.JSONLookup(getStructName(entity))
	if err != nil {
		panic(fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err))
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		panic(fmt.Errorf("type is not schema for entity %s", getStructName(entity)))
	}

	_ = schema.VisitJSON(defaultObject,
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.DefaultsSet(func() {
			if defaultSet != nil {
				defaultSet(&entity)
			}
		}),
	)

	b, _ := json.Marshal(defaultObject)
	if err = json.Unmarshal(b, &entity); err != nil {
		panic(fmt.Errorf("failed to new default entity %w", err))
	}
	return entity
}

// Validate validate the entity by OpenAPI schema defintion
func Validate[T any](entity T) error {
	val, err := schemas.JSONLookup(getStructName(entity))
	if err != nil {
		return fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err)
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		return fmt.Errorf("type is not schema for entity %s", getStructName(entity))
	}

	b, _ := json.Marshal(&entity)
	generic := make(map[string]interface{})
	_ = json.Unmarshal(b, &generic)

	err = schema.VisitJSON(generic,
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
	)
	switch err := err.(type) {
	case nil:
	case openapi3.MultiError:
		issues := openapi.ConvertError(err, "@body")
		jsonIssues := utils.ConvertJSONPaths(issues)
		return errs.NewValidateFieldsError(errs.ErrRequestValidate, jsonIssues)
	default:
		return err
	}
	return nil
}

// UnmarshalAndValidate unmarshal the request body as entity, combine the default value and validate it
func UnmarshalAndValidate[T any](r io.ReadCloser, entity *T, defaultSet func(*T)) error {
	objectMap := make(map[string]interface{})
	val, err := schemas.JSONLookup(getStructName(entity))
	if err != nil {
		panic(fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err))
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		panic(fmt.Errorf("type is not schema for entity %s", getStructName(entity)))
	}

	_ = schema.VisitJSON(objectMap,
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.DefaultsSet(func() {
			if defaultSet != nil {
				defaultSet(entity)
			}
		}),
	)

	b, _ := json.Marshal(objectMap)
	_ = json.Unmarshal(b, entity)

	if err := json.NewDecoder(r).Decode(&objectMap); err != nil {
		return err
	}

	err = schema.VisitJSON(objectMap,
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
	)
	switch err := err.(type) {
	case nil:
	case openapi3.MultiError:
		issues := openapi.ConvertError(err, "@body")
		jsonIssues := utils.ConvertJSONPaths(issues)
		return errs.NewValidateFieldsError(errs.ErrRequestValidate, jsonIssues)
	default:
		return err
	}
	b, _ = json.Marshal(objectMap)
	if err := json.Unmarshal(b, entity); err != nil {
		return errs.NewValidateError(err)
	}
	return nil
}

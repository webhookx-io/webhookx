package entities

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/utils"
	"io"
	"reflect"
)

var Schemas openapi3.Schemas

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
	Schemas = doc.Components.Schemas
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

func NewEntity[T any](defaultSet func(*T)) (T, map[string]interface{}) {
	var entity T
	jsonObject := make(map[string]interface{})
	val, err := Schemas.JSONLookup(getStructName(entity))
	if err != nil {
		panic(fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err))
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		panic(fmt.Errorf("type is not schema for entity %s", getStructName(entity)))
	}

	schema.VisitJSON(jsonObject,
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.DefaultsSet(func() {
			if defaultSet != nil {
				defaultSet(&entity)
			}
		}),
	)

	b, _ := json.Marshal(jsonObject)
	json.Unmarshal(b, &entity)
	return entity, jsonObject
}

func Validate[T any](entity T) error {
	val, err := Schemas.JSONLookup(getStructName(entity))
	if err != nil {
		return fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err)
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		return fmt.Errorf("type is not schema for entity %s", getStructName(entity))
	}

	b, _ := json.Marshal(entity)
	generic := make(map[string]interface{})
	json.Unmarshal(b, &generic)

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

func ValidateByMap[T any](entity *T, jsonObject map[string]interface{}) error {
	val, err := Schemas.JSONLookup(getStructName(entity))
	if err != nil {
		return fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err)
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		return fmt.Errorf("type is not schema for entity %s", getStructName(entity))
	}

	err = schema.VisitJSON(jsonObject,
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
	b, _ := json.Marshal(jsonObject)
	if err := json.Unmarshal(b, entity); err != nil {
		return errs.NewValidateError(err)
	}
	return nil
}

func UnmarshalAndValidate[T any](r io.ReadCloser, entity *T, defaultSet func(*T)) error {
	jsonObject := make(map[string]interface{})
	val, err := Schemas.JSONLookup(getStructName(entity))
	if err != nil {
		panic(fmt.Errorf("schema not found for entity %s: %w", getStructName(entity), err))
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		panic(fmt.Errorf("type is not schema for entity %s", getStructName(entity)))
	}

	schema.VisitJSON(jsonObject,
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.DefaultsSet(func() {
			if defaultSet != nil {
				defaultSet(entity)
			}
		}),
	)

	b, _ := json.Marshal(jsonObject)
	json.Unmarshal(b, entity)

	if err := json.NewDecoder(r).Decode(&jsonObject); err != nil {
		return err
	}

	err = schema.VisitJSON(jsonObject,
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
	b, _ = json.Marshal(jsonObject)
	if err := json.Unmarshal(b, entity); err != nil {
		return errs.NewValidateError(err)
	}
	return nil
}

package openapi

import (
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/utils"
	"reflect"
	"strings"
)

var schemas = make(openapi3.Schemas)

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

func SchemaVisitJSON(name string, generic map[string]interface{}, defaultSet func()) error {
	if generic == nil {
		generic = make(map[string]interface{})
	}

	val, err := schemas.JSONLookup(name)
	if err != nil {
		return fmt.Errorf("schema not found for entity %s: %w", name, err)
	}
	schema, ok := val.(*openapi3.Schema)
	if !ok {
		return fmt.Errorf("type is not schema for entity %s", name)
	}

	opts := []openapi3.SchemaValidationOption{
		openapi3.MultiErrors(),
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
	}

	if defaultSet != nil {
		opts = append(opts, openapi3.DefaultsSet(defaultSet))
	}

	err = schema.VisitJSON(generic, opts...)
	if defaultSet != nil {
		return nil
	}

	if err != nil {
		switch err := err.(type) {
		case openapi3.MultiError:
			issues := convertError(err, "@body")
			jsonIssues := utils.ConvertJSONPaths(issues)
			return errs.NewValidateFieldsError(errs.ErrRequestValidate, jsonIssues)
		default:
			return err
		}
	}

	return nil
}

// NewEntity new a default entity by OpenAPI schema definition
func NewEntity[T any](defaultSet func(*T)) T {
	var entity T
	var generic = make(map[string]interface{})
	entityName := GetStructName(entity)
	err := SchemaVisitJSON(entityName, generic, func() {
		if defaultSet != nil {
			defaultSet(&entity)
		}
	})

	if err != nil {
		panic(err)
	}

	b, _ := json.Marshal(&generic)
	if err := json.Unmarshal(b, &entity); err != nil {
		panic(fmt.Errorf("failed to new default entity %w", err))
	}
	return entity
}

// Validate validate the entity by OpenAPI schema defintion
func Validate[T any](entity T) error {
	entityName := GetStructName(entity)

	b, err := json.Marshal(&entity)
	if err != nil {
		return err
	}
	generic := make(map[string]interface{})
	err = json.Unmarshal(b, &generic)
	if err != nil {
		return err
	}

	return SchemaVisitJSON(entityName, generic, nil)
}

func GetStructName(i interface{}) string {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t.Name()
	}
	return ""
}

func convertError(me openapi3.MultiError, pathPrefix string) map[string][]interface{} {
	issues := make(map[string][]interface{})
	for _, err := range me {
		switch err := err.(type) {
		case *openapi3.SchemaError:
			field := pathPrefix
			if path := err.JSONPointer(); len(path) > 0 {
				field = fmt.Sprintf("%s.%s", field, strings.Join(path, "."))
			}
			issues[field] = append(issues[field], err.Reason)
		default:
			const unknown = "@unknown"
			issues[unknown] = append(issues[unknown], err.Error())
		}
	}
	return issues
}

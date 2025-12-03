package openapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/utils"
	"strings"
)

type FormatValidatorFunc[T any] func(T) error

func (fn FormatValidatorFunc[T]) Validate(value T) error { return fn(value) }

func init() {
	openapi3.DefineStringFormatValidator("jsonschema", FormatValidatorFunc[string](func(s string) error {
		schema := &openapi3.Schema{}
		if err := schema.UnmarshalJSON([]byte(s)); err != nil {
			return err
		}
		if len(schema.Extensions) == 0 || schema.Extensions["$schema"] == nil {
			if err := schema.Validate(context.TODO(), openapi3.EnableSchemaFormatValidation()); err != nil {
				return err
			}
			return nil
		}

		doc, err := jsonschema.UnmarshalJSON(strings.NewReader(s))
		if err != nil {
			return err
		}
		resourceFile := fmt.Sprintf("%x.json", utils.XXHash3(s))
		c := jsonschema.NewCompiler()
		err = c.AddResource(resourceFile, doc)
		var existErr *jsonschema.ResourceExistsError
		if err != nil && !errors.As(err, &existErr) {
			return err
		}
		_, err = c.Compile(resourceFile)
		if err != nil {
			if schemaValidateErr, ok := err.(*jsonschema.SchemaValidationError); ok {
				if validationErr, ok := schemaValidateErr.Err.(*jsonschema.ValidationError); ok {
					vErrs := errs.ParseJSONSchemaValidationError(validationErr, true)
					b, _ := json.Marshal(vErrs)
					return fmt.Errorf(`%s`, string(b))
				}
			}
			return err
		}
		return nil
	}))
}

func SetDefaults(schema *openapi3.Schema, defaults map[string]interface{}) error {
	data := make(map[string]interface{})
	_ = schema.VisitJSON(data,
		openapi3.MultiErrors(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.VisitAsRequest(),
		openapi3.DefaultsSet(func() {}),
	)

	// deep copy
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &defaults)
}

func Validate(schema *openapi3.Schema, value map[string]interface{}) error {
	err := schema.VisitJSON(value,
		openapi3.MultiErrors(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.VisitAsRequest(),
		openapi3.DefaultsSet(func() {}),
	)

	if err != nil {
		validateErr := errs.NewValidateError(errs.ErrRequestValidation)
		switch e := err.(type) {
		case openapi3.MultiError:
			handleMultiError(e, nil, validateErr.Fields)
		case *openapi3.SchemaError:
			handleMultiError(openapi3.MultiError{e}, nil, validateErr.Fields)
		default:
			validateErr.Message = err.Error()
		}
		errs.ConvertArrays(validateErr.Fields)
		return validateErr
	}

	return nil
}

func decodeMultiError(err error) openapi3.MultiError {
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		if me, ok := unwrapped.(openapi3.MultiError); ok {
			return me
		}
		return decodeMultiError(unwrapped)
	}
	return nil
}

func handleMultiError(me openapi3.MultiError, paths []string, fields map[string]interface{}) {
	for _, error := range me {
		switch e := error.(type) {
		case openapi3.MultiError:
			handleMultiError(e, paths, fields)
		case *openapi3.SchemaError:
			if e.SchemaField != "allOf" && e.SchemaField != "anyOf" && e.SchemaField != "oneOf" {
				errs.InsertError(fields, 0, append(paths, e.JSONPointer()...), &openapiSchemaError{e})
			}
			if decoded := decodeMultiError(e); decoded != nil {
				handleMultiError(decoded, e.JSONPointer(), fields)
			}
		default:
			const unknown = "@unknown"
			var unknowns []string
			if v, ok := fields[unknown]; !ok {
				unknowns = make([]string, 0)
			} else {
				unknowns = v.([]string)
			}
			fields[unknown] = append(unknowns, e.Error())
		}
	}
}

type openapiSchemaError struct {
	*openapi3.SchemaError
}

func (e *openapiSchemaError) FormatError() string {
	if e.SchemaField == "required" {
		return "required field missing"
	}
	return e.Reason
}

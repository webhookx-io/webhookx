package openapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/tidwall/gjson"
	"github.com/webhookx-io/webhookx/pkg/errs"
)

var (
	Spec *openapi3.T
)

type FormatValidatorFunc[T any] func(T) error

func (fn FormatValidatorFunc[T]) Validate(value T) error { return fn(value) }

func init() {
	openapi3.DefineStringFormatValidator("json", FormatValidatorFunc[string](func(s string) error {
		if !gjson.Valid(s) {
			return errors.New("not a valid JSON")
		}
		return nil
	}))
}

func ParseSpec(data []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	err = doc.Validate(
		loader.Context,
		openapi3.EnableSchemaFormatValidation(),
		openapi3.DisableSchemaDefaultsValidation(),
	)
	if err != nil {
		return nil, fmt.Errorf("OpenAPI document validation failed: %w", err)
	}
	return doc, nil
}

func LoadOpenAPI(data []byte) {
	doc, err := ParseSpec(data)
	if err != nil {
		panic(err)
	}
	Spec = doc
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
		convertArrays(validateErr.Fields)
		return validateErr
	}

	return nil
}

func ValidateParameters(r *http.Request, parameters openapi3.Parameters) error {
	options := openapi3filter.Options{
		MultiError:          true,
		SkipSettingDefaults: true,
	}
	options.WithCustomSchemaErrorFunc(formatError)
	input := &openapi3filter.RequestValidationInput{
		Request: r,
		Options: &options,
	}

	var me openapi3.MultiError
	for _, param := range parameters {
		if err := openapi3filter.ValidateParameter(context.TODO(), input, param.Value); err != nil {
			me = append(me, err)
		}
	}

	if len(me) > 0 {
		validateErr := errs.NewValidateError(errs.ErrRequestValidation)
		handleMultiError(me, nil, validateErr.Fields)
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
				p := append(paths, e.JSONPointer()...)
				switch e.SchemaField {
				case "discriminator":
					insertError(fields, 0, append(p, e.Schema.Discriminator.PropertyName), e)
				case "properties":
					re := regexp.MustCompile(`property "(.*?)" is unsupported`)
					if matches := re.FindStringSubmatch(e.Reason); len(matches) > 1 {
						p = append(p, matches[1])
					}
					insertError(fields, 0, p, e)
				default:
					insertError(fields, 0, p, e)
				}
			}
			if decoded := decodeMultiError(e); decoded != nil {
				handleMultiError(decoded, e.JSONPointer(), fields)
			}
		case *openapi3filter.RequestError:

			const params = "@params"
			var unknowns []string
			if v, ok := fields[params]; !ok {
				unknowns = make([]string, 0)
			} else {
				unknowns = v.([]string)
			}
			var msg string

			switch e.Err {
			case openapi3filter.ErrInvalidRequired:
				msg = fmt.Sprintf("%s: %s", e.Parameter.Name, "is required")
			case openapi3filter.ErrInvalidEmptyValue:
				msg = fmt.Sprintf("%s: %s", e.Parameter.Name, "is empty")
			default:
				msg = fmt.Sprintf("%s: %s", e.Parameter.Name, e.Err.Error())
			}
			fields[params] = append(unknowns, msg)
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

func convertArrays(m map[string]interface{}) {
	for k, v := range m {
		if val, ok := v.(map[string]interface{}); ok {
			if arr, ok := val[""].([]interface{}); ok && len(val) == 1 {
				m[k] = arr
				for _, arrv := range arr {
					if arrvalue, ok := arrv.(map[string]interface{}); ok {
						convertArrays(arrvalue)
					}
				}
			} else {
				convertArrays(val)
			}
		}
	}
}

func insertError(current map[string]interface{}, i int, paths []string, err *openapi3.SchemaError) {
	if len(paths) == 0 {
		current[""] = err.Reason
		return
	}

	key := paths[i]
	isIndex := false
	index := 0

	if i, err := strconv.Atoi(key); err == nil {
		isIndex = true
		index = i
	}

	if i == len(paths)-1 {
		// is last
		if isIndex {
			ensureArray(current, "", index)
			arr := current[""].([]interface{})
			arr[index] = formatError(err)
		} else {
			current[key] = formatError(err)
		}
		return
	}

	if isIndex {
		ensureArray(current, "", index)
		arr := current[""].([]interface{})
		if arr[index] == nil {
			arr[index] = make(map[string]interface{})
		}
		insertError(arr[index].(map[string]interface{}), i+1, paths, err)
	} else {
		if current[key] == nil {
			current[key] = make(map[string]interface{})
		}
		insertError(current[key].(map[string]interface{}), i+1, paths, err)
	}
}

func ensureArray(m map[string]interface{}, key string, index int) {
	if val, ok := m[key]; ok {
		if arr, ok := val.([]interface{}); ok && len(arr) > index {
			return
		}
	}

	newArr := make([]interface{}, index+1)
	if old, ok := m[key].([]interface{}); ok {
		copy(newArr, old)
	}
	m[key] = newArr
}

func formatError(e *openapi3.SchemaError) string {
	switch e.SchemaField {
	case "required":
		return "required field missing"
	case "properties":
		return "property is unsupported"
	case "discriminator":
		enums := slices.Sorted(maps.Keys(e.Schema.Discriminator.Mapping))
		if len(enums) > 0 {
			allowedValues, _ := json.Marshal(enums)
			return fmt.Sprintf("value is not one of the allowed values %s", string(allowedValues))
		}
		return e.Reason
	default:
		return e.Reason
	}
}

package openapi

import (
	"encoding/json"
	"errors"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"strconv"
)

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
		// openapi3.SetSchemaErrorMessageCustomizer(customizeMessageErrorfunc),
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

func decodeMultiError(err error) openapi3.MultiError {
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		if me, ok := unwrapped.(openapi3.MultiError); ok {
			return me
		}
		return decodeMultiError(unwrapped)
	}
	return nil
}

func handleMultiError(me openapi3.MultiError, parents []string, fields map[string]interface{}) {
	for _, error := range me {
		switch e := error.(type) {
		case openapi3.MultiError:
			handleMultiError(e, parents, fields)
		case *openapi3.SchemaError:
			if e.SchemaField != "allOf" && e.SchemaField != "anyOf" && e.SchemaField != "oneOf" {
				insertError(fields, 0, append(parents, e.JSONPointer()...), e)
			}
			if decoded := decodeMultiError(e); decoded != nil {
				handleMultiError(decoded, e.JSONPointer(), fields)
			}
		default:
			// ???
			//const unknown = "@unknown"
			//issues[unknown] = append(issues[unknown], err.Error())
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
			if err.Origin == nil {
				arr[index] = formatError(err)
			}
		} else {
			if err.Origin == nil {
				current[key] = formatError(err)
			}
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
	if e.SchemaField == "required" {
		return "required field missing"
	}
	return e.Reason
}

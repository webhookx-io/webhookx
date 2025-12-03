package errs

import (
	"encoding/json"
	"fmt"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strconv"
	"strings"
)

type SchemaError interface {
	FormatError() string
}

func ConvertArrays(m map[string]interface{}) {
	for k, v := range m {
		if val, ok := v.(map[string]interface{}); ok {
			if arr, ok := val[""].([]interface{}); ok && len(val) == 1 {
				m[k] = arr
				for _, arrv := range arr {
					if arrvalue, ok := arrv.(map[string]interface{}); ok {
						ConvertArrays(arrvalue)
					}
				}
			} else {
				ConvertArrays(val)
			}
		}
	}
}

func InsertError(current map[string]interface{}, i int, paths []string, err SchemaError) {
	if len(paths) == 0 {
		current[""] = err.FormatError()
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
			arr[index] = err.FormatError()
		} else {
			if current[key] == nil {
				current[key] = err.FormatError()
			} else {
				current[key] = fmt.Sprintf(`%v or %s`, current[key], err.FormatError())
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
		InsertError(arr[index].(map[string]interface{}), i+1, paths, err)
	} else {
		if current[key] == nil {
			current[key] = make(map[string]interface{})
		}
		InsertError(current[key].(map[string]interface{}), i+1, paths, err)
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

// ParseValidationError analyze ValidationError and convert to map
// the key is the field,
// the value is the format error, or nested map
func ParseJSONSchemaValidationError(err *jsonschema.ValidationError) map[string]interface{} {
	errorMap := make(map[string]interface{})

	var collectErrors func(e *jsonschema.ValidationError, paths []string, fields map[string]interface{})
	collectErrors = func(e *jsonschema.ValidationError, paths []string, fields map[string]interface{}) {
		if len(e.Causes) > 0 {
			switch e.ErrorKind.(type) {
			case *kind.OneOf, *kind.AnyOf, *kind.AllOf, *kind.Contains, *kind.PropertyNames, *kind.DependentRequired:
				// skip causes
				if len(e.InstanceLocation) > 0 {
					InsertError(fields, 0, e.InstanceLocation, &jsonSchemaError{e})
				} else {
					InsertError(fields, 0, append(paths, e.ErrorKind.KeywordPath()...), &jsonSchemaError{e})
				}
			default:
				for _, cause := range e.Causes {
					collectErrors(cause, append(paths, e.InstanceLocation...), fields)
				}
			}
		} else {
			switch e.ErrorKind.(type) {
			case *kind.Schema:
				// skip root schema error
			case *kind.Required:
				kindErr := e.ErrorKind.(*kind.Required)
				for _, missing := range kindErr.Missing {
					InsertError(fields, 0, append(paths, missing), &jsonSchemaError{e})
				}
			default:
				if len(e.InstanceLocation) > 0 {
					InsertError(fields, 0, e.InstanceLocation, &jsonSchemaError{e})
				} else {
					InsertError(fields, 0, append(paths, e.ErrorKind.KeywordPath()...), &jsonSchemaError{e})
				}
			}
		}
	}
	collectErrors(err, nil, errorMap)
	ConvertArrays(errorMap)
	return errorMap
}

type jsonSchemaError struct {
	*jsonschema.ValidationError
}

func (e *jsonSchemaError) reason() string {
	return e.ErrorKind.LocalizedString(message.NewPrinter(language.English))
}

// format the error same as openapi3 schema error Reason
func (e *jsonSchemaError) FormatError() string {
	switch eKind := e.ErrorKind.(type) {
	case *kind.Required:
		return "required field missing"
	case *kind.Type:
		// Format type errors to match OpenAPI3's format: "value must be an integer", "value must be a string", etc.
		if len(eKind.Want) == 1 {
			typeName := eKind.Want[0]
			article := "a"
			if typeName == "integer" || typeName == "array" || typeName == "object" {
				article = "an"
			}
			return fmt.Sprintf("value must be %s %s", article, typeName)
		}
		return fmt.Sprintf("value must be one of: %s", strings.Join(eKind.Want, ", "))
	case *kind.Enum:
		// Format enum errors to match OpenAPI3's format
		enumValues, _ := json.Marshal(eKind.Want)
		return fmt.Sprintf("value is not one of the allowed values %s", string(enumValues))
	case *kind.Const:
		constValue, _ := json.Marshal(eKind.Want)
		return fmt.Sprintf("value must be equal to %s", string(constValue))
	case *kind.Minimum:
		want, _ := eKind.Want.Float64()
		return fmt.Sprintf("number must be at least %g", want)
	case *kind.Maximum:
		want, _ := eKind.Want.Float64()
		return fmt.Sprintf("number must be at most %g", want)
	case *kind.ExclusiveMinimum:
		want, _ := eKind.Want.Float64()
		return fmt.Sprintf("number must be more than %g", want)
	case *kind.ExclusiveMaximum:
		want, _ := eKind.Want.Float64()
		return fmt.Sprintf("number must be less than %g", want)
	case *kind.MultipleOf:
		want, _ := eKind.Want.Float64()
		return fmt.Sprintf("number must be a multiple of %g", want)
	case *kind.MinLength:
		return fmt.Sprintf("minimum string length is %d", eKind.Want)
	case *kind.MaxLength:
		return fmt.Sprintf("maximum string length is %d", eKind.Want)
	case *kind.Pattern:
		return fmt.Sprintf(`string doesn't match the regular expression "%s"`, eKind.Want)
	case *kind.Format:
		return fmt.Sprintf(`string doesn't match the format %q`, eKind.Want)
	case *kind.MinItems:
		return fmt.Sprintf("minimum number of items is %d", eKind.Want)
	case *kind.MaxItems:
		return fmt.Sprintf("maximum number of items is %d", eKind.Want)
	case *kind.UniqueItems:
		return "duplicate items found"
	case *kind.Contains:
		return "no items match contains schema"
	case *kind.MinContains:
		if len(eKind.Got) == 0 {
			return fmt.Sprintf("min %d items required to match contains schema, but none matched", eKind.Want)
		}
		return fmt.Sprintf("min %d items required to match contains schema, but matched %d items", eKind.Want, len(eKind.Got))
	case *kind.MaxContains:
		return fmt.Sprintf("max %d items required to match contains schema, but matched %d items", eKind.Want, len(eKind.Got))
	case *kind.MinProperties:
		return fmt.Sprintf("there must be at least %d properties", eKind.Want)
	case *kind.MaxProperties:
		return fmt.Sprintf("there must be at most %d properties", eKind.Want)
	case *kind.AdditionalItems:
		return fmt.Sprintf("last %d additionalItem(s) not allowed", eKind.Count)
	case *kind.AdditionalProperties:
		return fmt.Sprintf("property %q is unsupported", eKind.Properties[0])
	case *kind.PropertyNames:
		return fmt.Sprintf("invalid propertyName %q", eKind.Property)
	case *kind.Dependency:
		return fmt.Sprintf("properties %s required, if %s exists", strings.Join(eKind.Missing, ", "), eKind.Prop)
	case *kind.DependentRequired:
		return fmt.Sprintf("properties %s required, if %s exists", strings.Join(eKind.Missing, ", "), eKind.Prop)
	case *kind.OneOf:
		if len(eKind.Subschemas) == 0 {
			return `value doesn't match any schema from "oneOf"`
		}
		return fmt.Sprintf(`value matches more than one schema from "oneOf" (matches schemas at indices %v)`, eKind.Subschemas)
	case *kind.AnyOf:
		return `value doesn't match any schema from "anyOf"`
	case *kind.AllOf:
		return `value doesn't match all schemas from "allOf"`
	case *kind.Not:
		return "value matches a forbidden schema"
	}
	return e.reason()
}

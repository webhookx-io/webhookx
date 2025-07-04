package openapi

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func convertError(me openapi3.MultiError) map[string][]string {
	issues := make(map[string][]string)
	for _, err := range me {
		const prefixBody = "@body"
		switch err := err.(type) {
		case *openapi3.SchemaError:
			// Can inspect schema validation errors here, e.g. err.Value
			field := prefixBody
			if path := err.JSONPointer(); len(path) > 0 {
				field = fmt.Sprintf("%s.%s", field, strings.Join(path, "."))
			}
			issues[field] = append(issues[field], err.Reason)
		case *openapi3filter.RequestError: // possible there were multiple issues that failed validation

			// check if invalid HTTP parameter
			if err.Parameter != nil {
				prefix := err.Parameter.In
				name := fmt.Sprintf("@%s.%s", prefix, err.Parameter.Name)
				// issues[name] = append(issues[name], err.Error())
				errs := ExtractErrors(err.Err, name)
				for k, v := range errs {
					issues[k] = append(issues[k], v...)
				}
				continue
			}

			if err, ok := err.Err.(openapi3.MultiError); ok {
				for k, v := range convertError(err) {
					issues[k] = append(issues[k], v...)
				}
				continue
			}

			// check if requestBody
			if err.RequestBody != nil {
				issues[prefixBody] = append(issues[prefixBody], err.Error())
				continue
			}
		default:
			const unknown = "@unknown"
			issues[unknown] = append(issues[unknown], err.Error())
		}
	}
	return issues
}

func ExtractErrors(err error, field string) map[string][]string {
	if me, ok := err.(openapi3.MultiError); ok {
		for _, e := range me {
			// check e is schema error
			if se, ok := e.(*openapi3.SchemaError); ok {
				if path := se.JSONPointer(); len(path) > 0 {
					field = fmt.Sprintf("%s.%s", field, strings.Join(path, "."))
				}
				return map[string][]string{field: {se.Reason}}
			} else {
				continue
			}
		}
	}
	return map[string][]string{}
}

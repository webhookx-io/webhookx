package openapi

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"strings"
)

func ConvertError(me openapi3.MultiError, pathPrefix string) map[string][]interface{} {
	issues := make(map[string][]interface{})
	for _, err := range me {
		switch err := err.(type) {
		case *openapi3.SchemaError:
			field := pathPrefix
			if path := err.JSONPointer(); len(path) > 0 {
				field = fmt.Sprintf("%s.%s", field, strings.Join(path, "."))
			}
			issues[field] = append(issues[field], err.Reason)
		case *openapi3filter.RequestError:
			if err.Parameter != nil {
				prefix := err.Parameter.In
				name := fmt.Sprintf("@%s.%s", prefix, err.Parameter.Name)
				if se, ok := err.Err.(openapi3.MultiError); ok {
					errs := ConvertError(se, name)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				}
				continue
			}

			if err, ok := err.Err.(openapi3.MultiError); ok {
				for k, v := range ConvertError(err, pathPrefix) {
					issues[k] = append(issues[k], v...)
				}
				continue
			}

			if err.RequestBody != nil {
				if se, ok := err.Err.(openapi3.MultiError); ok {
					errs := ConvertError(se, pathPrefix)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				} else {
					errs := ConvertError(openapi3.MultiError{err.Err}, pathPrefix)
					for k, v := range errs {
						issues[k] = append(issues[k], v...)
					}
				}
				continue
			}
		default:
			const unknown = "@unknown"
			issues[unknown] = append(issues[unknown], err.Error())
		}
	}
	return issues
}

package openapi

import (
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
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
		default:
			const unknown = "@unknown"
			issues[unknown] = append(issues[unknown], err.Error())
		}
	}
	return issues
}

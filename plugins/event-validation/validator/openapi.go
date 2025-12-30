package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type OpenApiValidator struct {
	schema *openapi3.Schema
}

func NewOpenApiValidator(schemaDef string) (*OpenApiValidator, error) {
	schema := openapi3.NewSchema()
	err := schema.UnmarshalJSON([]byte(schemaDef))
	if err != nil {
		return nil, err
	}
	return &OpenApiValidator{schema: schema}, nil
}

func (v *OpenApiValidator) Validate(value interface{}) error {
	err := v.schema.VisitJSON(value,
		openapi3.MultiErrors(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.VisitAsRequest(),
	)

	if err != nil {
		var validateError ValidateError
		var walk func(me openapi3.MultiError)

		walk = func(me openapi3.MultiError) {
			for _, err := range me {
				switch e := err.(type) {
				case openapi3.MultiError:
					walk(e)
				case *openapi3.SchemaError:
					validateError = append(validateError, errors.New(formatError(e)))
				default:
					validateError = append(validateError, e)
				}
			}
		}
		switch e := err.(type) {
		case openapi3.MultiError:
			walk(e)
		case *openapi3.SchemaError:
			walk(openapi3.MultiError{e})
		default:
			validateError = append(validateError, e)
		}
		return validateError
	}
	return nil
}

func formatError(e *openapi3.SchemaError) string {
	return fmt.Sprintf("at '%s': %s", strings.Join(e.JSONPointer(), "/"), e.Reason)
}

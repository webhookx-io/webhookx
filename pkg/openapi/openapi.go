package openapi

import (
	"encoding/json"
	"errors"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/errs"
)

var validationErr = errors.New("request validation") // TODO duplicated

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
		openapi3.VisitAsRequest(),
		openapi3.DisableReadOnlyValidation(),
		// openapi3.SetSchemaErrorMessageCustomizer(customizeMessageErrorfunc),
	)
	if err != nil {
		validateErr := errs.NewValidateError(validationErr)
		for _, e := range err.(openapi3.MultiError) {
			switch se := e.(type) {
			case *openapi3.SchemaError:
				node := validateErr.Fields
				fields := se.JSONPointer()
				for i, field := range fields {
					if i < len(fields)-1 {
						if node[field] == nil {
							node[field] = make(map[string]interface{})
						}
						node = node[field].(map[string]interface{})
					} else {
						node[field] = formatError(se)
					}
				}
			default:
				// TODO ???
				//const unknown = "@unknown"
				//issues[unknown] = append(issues[unknown], err.Error())
			}
		}
		return validateErr
	}

	return nil
}

func formatError(e *openapi3.SchemaError) string {
	if e.SchemaField == "required" {
		return "required field missing"
	}
	return e.Reason
}

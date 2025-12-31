package validator

import (
	"errors"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type JsonSchemaValidator struct {
	schema *jsonschema.Schema
}

func NewJsonSchemaValidator(draft *jsonschema.Draft, schemaDef string) (*JsonSchemaValidator, error) {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(draft)
	schemaJson, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaDef))
	if err != nil {
		return nil, err
	}
	err = c.AddResource("schema.json", schemaJson)
	if err != nil {
		return nil, err
	}
	schema, err := c.Compile("schema.json")
	if err != nil {
		return nil, err
	}

	validator := &JsonSchemaValidator{
		schema: schema,
	}
	return validator, nil
}

func (v *JsonSchemaValidator) Validate(value interface{}) error {
	err := v.schema.Validate(value)
	return convertError(err)
}

func convertError(err error) error {
	var e *jsonschema.ValidationError
	if !errors.As(err, &e) {
		return err
	}

	var validateError ValidateError
	var walk func(e *jsonschema.ValidationError)

	walk = func(e *jsonschema.ValidationError) {
		if len(e.Causes) == 0 {
			validateError = append(validateError, e)
			return
		}
		for _, cause := range e.Causes {
			walk(cause)
		}
	}

	walk(e)
	return validateError
}

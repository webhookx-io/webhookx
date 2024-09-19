package utils

import (
	"errors"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"reflect"
	"strings"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

var validationErr = errors.New("request validation")

func Validate(v interface{}) error {
	err := validate.Struct(v)
	if err != nil {
		validateErr := errs.NewValidateError(validationErr)
		t := reflect.ValueOf(v).Type()
		for _, e := range err.(validator.ValidationErrors) {
			fields := strings.Split(e.StructNamespace(), ".")
			node := validateErr.Fields
			parentT := t
			for i := 1; i < len(fields); i++ {
				f, ok := getField(parentT, fields[i])
				if !ok {
					continue
				}

				fieldName := fieldName(f)
				if i < len(fields)-1 {
					if node[fieldName] == nil {
						node[fieldName] = make(map[string]interface{})
					}
					node = node[fieldName].(map[string]interface{})
				} else {
					node[fieldName] = formatError(e)
				}
				parentT = f.Type
			}
		}
		return validateErr
	}
	return nil
}

func formatError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "required field missing"
	case "oneof":
		return fmt.Sprintf("invalid value: %s", fe.Value())
	case "gt":
		return fmt.Sprintf("value must be > %s", fe.Param())
	case "gte":
		return fmt.Sprintf("value must be >= %s", fe.Param())
	case "lt":
		return fmt.Sprintf("value must be < %s", fe.Param())
	case "lte":
		return fmt.Sprintf("value must be <= %s", fe.Param())
	case "min":
		return fmt.Sprintf("length must be at least %s", fe.Param())
	}
	return fe.Error()
}

func fieldName(field reflect.StructField) string {
	name := field.Tag.Get("json")
	if name == "" {
		name = field.Name
	}
	return name
}

func getField(t reflect.Type, field string) (reflect.StructField, bool) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.FieldByName(field)
}

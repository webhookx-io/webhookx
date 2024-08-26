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

func Validate(v interface{}) error {
	err := validate.Struct(v)
	if err != nil {
		validateErr := errs.NewValidateError(errors.New("request validation"))
		for _, e := range err.(validator.ValidationErrors) {
			// todo nested fields
			t := reflect.ValueOf(v).Type()
			list := strings.Split(e.StructNamespace(), ".")
			list = list[1:]
			field := findField(t, list)
			msg := fmt.Sprintf("invalid %s", field.Tag.Get("json"))
			validateErr.Fields[field.Tag.Get("json")] = msg
		}
		return validateErr
	}
	return nil
}

func findField(s reflect.Type, path []string) *reflect.StructField {
	var field reflect.StructField

	t := s
	for _, p := range path {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}

		f, ok := t.FieldByName(p)
		if ok {
			field = f
			t = f.Type
		}
	}

	return &field
}

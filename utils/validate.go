package utils

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"reflect"
	"regexp"
	"strings"
	"sync"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

var mux sync.RWMutex
var formatters = make(map[string]func(fe validator.FieldError) string)

func init() {
	RegisterFormatter("required", func(fe validator.FieldError) string {
		return "required field missing"
	})
	RegisterFormatter("oneof", func(fe validator.FieldError) string {
		return fmt.Sprintf("invalid value: %s", fe.Value())
	})
	RegisterFormatter("gt", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be > %s", fe.Param())
	})
	RegisterFormatter("gte", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be >= %s", fe.Param())
	})
	RegisterFormatter("lt", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be < %s", fe.Param())
	})
	RegisterFormatter("lte", func(fe validator.FieldError) string {
		return fmt.Sprintf("value must be <= %s", fe.Param())
	})
	RegisterFormatter("min", func(fe validator.FieldError) string {
		return fmt.Sprintf("length must be at least %s", fe.Param())
	})
	RegisterFormatter("max", func(fe validator.FieldError) string {
		return fmt.Sprintf("length must be at most %s", fe.Param())
	})
	RegisterFormatter("url", func(fe validator.FieldError) string {
		return "value must be a valid url"
	})
	RegisterFormatter("file", func(fe validator.FieldError) string {
		return "value must be a valid exist file"
	})
	RegisterFormatter("json", func(fe validator.FieldError) string {
		return "value must be a valid json string"
	})
}

func RegisterValidation(tag string, fn validator.Func) {
	err := validate.RegisterValidation(tag, fn)
	if err != nil {
		panic(err)
	}
}

func RegisterFormatter(tag string, fn func(fe validator.FieldError) string) {
	mux.Lock()
	defer mux.Unlock()
	formatters[tag] = fn
}

const fieldPlacehoder = "#field%d"

func Validate(v interface{}) error {
	err := validate.Struct(v)
	if err != nil {
		validateErr := errs.NewValidateError(errs.ErrRequestValidation)
		for _, e := range err.(validator.ValidationErrors) {
			namespace := e.Namespace()
			placeholders := make(map[string]string)
			if strings.ContainsAny(namespace, "[]") {
				re := regexp.MustCompile(`\w+\[[^\]]+\]`)
				matches := re.FindAllString(namespace, -1)
				for i, field := range matches {
					idx := fmt.Sprintf(fieldPlacehoder, i)
					placeholders[idx] = field
					namespace = strings.Replace(namespace, field, idx, 1)
				}
			}
			fields := strings.Split(namespace, ".")
			node := validateErr.Fields
			for i := 1; i < len(fields); i++ {
				fieldName := fields[i]
				if actualField, ok := placeholders[fieldName]; ok {
					fieldName = actualField
				}
				if i < len(fields)-1 {
					if node[fieldName] == nil {
						node[fieldName] = make(map[string]interface{})
					}
					node = node[fieldName].(map[string]interface{})
				} else {
					node[fieldName] = formatError(e)
				}
			}
		}
		return validateErr
	}
	return nil
}

func formatError(fe validator.FieldError) string {
	mux.RLock()
	defer mux.RUnlock()
	if formatter, ok := formatters[fe.Tag()]; ok {
		return formatter(fe)
	}
	return fe.Error()
}

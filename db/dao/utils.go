package dao

import (
	"errors"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/webhookx-io/webhookx/utils"
)

// EachField traverse each database field
func EachField(entity interface{}, fn func(field reflect.StructField, value reflect.Value, column string)) {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		column := utils.DefaultIfZero(field.Tag.Get("db"), strings.ToLower(field.Name))
		if column == "-" {
			continue
		}
		if field.Anonymous {
			EachField(value.Interface(), fn)
		} else {
			fn(field, value, column)
		}
	}
}

func is23505(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

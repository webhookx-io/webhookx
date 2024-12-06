package utils

import (
	"reflect"
)

func DefaultIfZero[T any](v T, fallback T) T {
	if reflect.ValueOf(v).IsZero() {
		return fallback
	}
	return v
}

package utils

import "time"

func Pointer[T any](v T) *T {
	return &v
}

func PointerValue[T any](v *T) T {
	if v == nil {
		return *new(T)
	}
	return *v
}

func DurationS(seconds int64) time.Duration {
	return time.Duration(seconds) * time.Second
}

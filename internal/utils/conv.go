package utils

func Pointer[T any](v T) *T {
	return &v
}

func PointerValue[T any](v *T) T {
	if v == nil {
		return *new(T)
	}
	return *v
}

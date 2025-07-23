package utils

import (
	"encoding/json"
	"time"
)

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

func StructToMap(v interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

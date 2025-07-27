package utils

import (
	"encoding/json"
	"reflect"
)

func DefaultIfZero[T any](v T, fallback T) T {
	if reflect.ValueOf(v).IsZero() {
		return fallback
	}
	return v
}

func MergeMap(dst, src map[string]interface{}) {
	for k, v := range src {
		if srcv, ok := v.(map[string]interface{}); ok {
			if dstv, ok := dst[k].(map[string]interface{}); ok {
				MergeMap(dstv, srcv)
			} else {
				dst[k] = srcv
			}
		} else {
			dst[k] = v
		}
	}
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

func MapToStruct(data map[string]interface{}, v interface{}) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

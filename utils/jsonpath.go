package utils

import (
	"strconv"
	"strings"
)

func mergePath(root map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")

	current := root
	for i, part := range parts {
		isLast := i == len(parts)-1

		if idx, err := strconv.Atoi(part); err == nil {
			arr, ok := current[""].([]interface{})
			if !ok {
				arr = make([]interface{}, idx+1)
				current[""] = arr
			}

			if len(arr) <= idx {
				newArr := make([]interface{}, idx+1)
				copy(newArr, arr)
				arr = newArr
				current[""] = arr
			}

			if isLast {
				arr[idx] = value
			} else {
				m, ok := arr[idx].(map[string]interface{})
				if !ok {
					m = make(map[string]interface{})
					arr[idx] = m
				}
				current = m
			}
		} else {
			if isLast {
				current[part] = value
			} else {
				next, ok := current[part].(map[string]interface{})
				if !ok {
					next = make(map[string]interface{})
					current[part] = next
				}
				current = next
			}
		}
	}
}

func ConvertJSONPaths(input map[string][]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for path, val := range input {
		idx := strings.Index(path, ".")
		if idx == -1 {
			result[path] = val
			continue
		}
		prefix := path[:idx]
		subPath := path[idx+1:]

		prefixMap, ok := result[prefix].(map[string]interface{})
		if !ok {
			prefixMap = make(map[string]interface{})
			result[prefix] = prefixMap
		}

		mergePath(prefixMap, subPath, val)
	}

	convertArrays(result)

	return result
}

func convertArrays(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			if arr, ok := val[""].([]interface{}); ok && len(val) == 1 {
				m[k] = arr
			} else {
				convertArrays(val)
			}
		}
	}
}

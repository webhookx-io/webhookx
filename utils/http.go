package utils

import (
	"encoding/json"
	"net/http"
	"strings"
)

func JsonResponse(code int, w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(bytes)
}

func HeaderMap(header http.Header) map[string]string {
	headers := make(map[string]string)
	for name, values := range header {
		headers[name] = strings.Join(values, ",")
	}
	return headers
}

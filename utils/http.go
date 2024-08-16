package utils

import (
	"encoding/json"
	"net/http"
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

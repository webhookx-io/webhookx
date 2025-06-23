package utils

import (
	"net/http"
	"strings"
)

func HeaderMap(header http.Header) map[string]string {
	headers := make(map[string]string)
	for name, values := range header {
		headers[name] = strings.Join(values, ",")
	}
	return headers
}

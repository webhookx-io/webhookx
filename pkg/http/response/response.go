package response

import (
	"encoding/json"
	"net/http"

	"github.com/webhookx-io/webhookx/constants"
	"go.uber.org/zap"
)

func JSON(w http.ResponseWriter, code int, data interface{}) {
	_json(w, code, data, false)
}

func _json(w http.ResponseWriter, code int, data interface{}, pretty bool) {
	headers := map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}

	var bytes []byte
	switch v := data.(type) {
	case string:
		bytes = []byte(v)
	default:
		var err error
		if pretty {
			bytes, err = json.MarshalIndent(data, "", "  ")
		} else {
			bytes, err = json.Marshal(data)
		}
		if err != nil {
			panic(err)
		}
	}

	Response(w, headers, code, bytes)
}

func Text(w http.ResponseWriter, code int, body string) {
	headers := map[string]string{
		"Content-Type": "text/plain",
	}
	Response(w, headers, code, []byte(body))
}

func Response(w http.ResponseWriter, headers map[string]string, code int, body []byte) {
	for _, header := range constants.DefaultResponseHeaders {
		w.Header().Set(header.Name, header.Value)
	}

	for k, v := range headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(code)

	if body != nil {
		_, err := w.Write(body)
		if err != nil {
			zap.S().Error("response write error", zap.Error(err))
		}
	}
}

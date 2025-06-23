package response

import (
	"encoding/json"
	"github.com/webhookx-io/webhookx/constants"
	"net/http"
)

func JSON(w http.ResponseWriter, code int, data interface{}) {
	_json(w, code, data, false)
}

func _json(w http.ResponseWriter, code int, data interface{}, pretty bool) {
	for _, header := range constants.DefaultResponseHeaders {
		w.Header().Set(header.Name, header.Value)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(code)

	if data == nil {
		return
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
	_, err := w.Write(bytes)
	if err != nil {
		panic(err)
	}
}

func Text(w http.ResponseWriter, code int, body string) {
	for _, header := range constants.DefaultResponseHeaders {
		w.Header().Set(header.Name, header.Value)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	_, err := w.Write([]byte(body))
	if err != nil {
		panic(err)
	}
}

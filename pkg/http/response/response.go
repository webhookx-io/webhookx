package response

import (
	"encoding/json"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/constants"
	"net/http"
)

type Header struct {
	Name  string
	Value string
}

var (
	// TODO
	DefaultResponseHeaders = []Header{
		{Name: "Server", Value: "WebhookX/" + config.VERSION},
	}
)

func JSON(w http.ResponseWriter, code int, data interface{}) {
	for header, value := range constants.DefaultResponseHeaders {
		w.Header().Set(header, value)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(code)

	if data == nil {
		return
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(bytes)
}

func Text(w http.ResponseWriter, code int, body string) {
	for header, value := range constants.DefaultResponseHeaders {
		w.Header().Set(header, value)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(body))
}

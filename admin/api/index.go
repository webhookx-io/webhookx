package api

import (
	"github.com/webhookx-io/webhookx/config"
	"net/http"
)

type IndexResponse struct {
	Version       string         `json:"version"`
	Message       string         `json:"message"`
	Configuration *config.Config `json:"configuration"`
}

func (api *API) Index(w http.ResponseWriter, r *http.Request) {
	var response IndexResponse

	response.Version = config.VERSION
	response.Message = "Welcome to WebhookX"
	response.Configuration = api.cfg

	api.json(200, w, response)
}

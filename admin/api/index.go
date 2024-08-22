package api

import (
	"github.com/webhookx-io/webhookx/config"
	"net/http"
)

func (api *API) Index(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})

	data["version"] = config.VERSION

	api.json(200, w, data)
}

package api

import (
	"net/http"

	"github.com/webhookx-io/webhookx/pkg/license"
)

func (api *API) GetLicense(w http.ResponseWriter, r *http.Request) {
	api.json(200, w, license.GetLicenser().License())
}

package api

import (
	"errors"
	"net/http"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) PagePlugin(w http.ResponseWriter, r *http.Request) {
	var q query.PluginQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	list, total, err := api.db.PluginsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetPlugin(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	plugin, err := api.db.PluginsWS.Get(r.Context(), id)
	api.assert(err)

	if plugin == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, plugin)
}

func (api *API) CreatePlugin(w http.ResponseWriter, r *http.Request) {
	var model entities.Plugin
	defaults := map[string]interface{}{"id": utils.KSUID()}
	if err := ValidateRequest(r, defaults, &model); err != nil {
		api.error(400, w, err)
		return
	}

	if err := model.Validate(); err != nil {
		var licenseError *errs.LicenseError
		if errors.As(err, &licenseError) {
			api.error(403, w, err)
		} else {
			api.error(400, w, err)
		}
		return
	}

	err := api.db.PluginsWS.Insert(r.Context(), &model)
	api.assert(err)

	api.json(201, w, model)
}

func (api *API) UpdatePlugin(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	model, err := api.db.PluginsWS.Get(r.Context(), id)
	api.assert(err)
	if model == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	defaults := utils.Must(utils.StructToMap(model))
	if err := ValidateRequest(r, defaults, model); err != nil {
		api.error(400, w, err)
		return
	}

	if err := model.Validate(); err != nil {
		var licenseError *errs.LicenseError
		if errors.As(err, &licenseError) {
			api.error(403, w, err)
		} else {
			api.error(400, w, err)
		}
		return
	}

	model.ID = id
	err = api.db.PluginsWS.Update(r.Context(), model)
	api.assert(err)

	api.json(200, w, model)
}

func (api *API) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	_, err := api.db.PluginsWS.Delete(r.Context(), id)
	api.assert(err)

	w.WriteHeader(204)
}

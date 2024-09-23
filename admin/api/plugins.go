package api

import (
	"encoding/json"
	"errors"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
)

func (api *API) PagePlugin(w http.ResponseWriter, r *http.Request) {
	var q query.PluginQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	list, total, err := api.DB.PluginsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetPlugin(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	Plugin, err := api.DB.PluginsWS.Get(r.Context(), id)
	api.assert(err)

	if Plugin == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, Plugin)
}

func (api *API) CreatePlugin(w http.ResponseWriter, r *http.Request) {
	var model entities.Plugin
	model.Init()
	api.assert(defaults.Set(&model))
	if err := json.NewDecoder(r.Body).Decode(&model); err != nil {
		api.error(400, w, err)
		return
	}

	if err := model.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	p := plugin.New(model.Name)
	if p == nil {
		api.error(400, w, errors.New("unknown plugin name"))
		return
	}

	config := p.Config()
	if model.Config != nil {
		err := json.Unmarshal(model.Config, config)
		if err != nil {
			api.error(400, w, err)
			return
		}
	}
	config.ProcessDefault()
	if err := config.Validate(); err != nil {
		api.error(400, w, err)
		return
	}
	model.Config = utils.Must(json.Marshal(config))

	err := api.DB.PluginsWS.Insert(r.Context(), &model)
	api.assert(err)

	api.json(201, w, model)
}

func (api *API) UpdatePlugin(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	model, err := api.DB.PluginsWS.Get(r.Context(), id)
	api.assert(err)
	if model == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		api.error(400, w, err)
		return
	}

	if err := model.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	p := plugin.New(model.Name)
	if p == nil {
		api.error(400, w, errors.New("unknown plugin name"))
		return
	}

	config := p.Config()
	if model.Config != nil {
		err := json.Unmarshal(model.Config, config)
		if err != nil {
			api.error(400, w, err)
			return
		}
	}
	config.ProcessDefault()
	if err := config.Validate(); err != nil {
		api.error(400, w, err)
		return
	}
	model.Config = utils.Must(json.Marshal(config))

	model.ID = id
	err = api.DB.PluginsWS.Update(r.Context(), model)
	api.assert(err)

	api.json(200, w, model)
}

func (api *API) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	_, err := api.DB.PluginsWS.Delete(r.Context(), id)
	api.assert(err)

	w.WriteHeader(204)
}

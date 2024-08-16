package api

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"net/http"
)

func (api *API) PageSource(w http.ResponseWriter, r *http.Request) {
	var q query.SourceQuery
	api.bindQuery(r, &q.Query)
	list, total, err := api.DB.SourcesWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetSource(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	endpoint, err := api.DB.SourcesWS.Get(r.Context(), id)
	api.assert(err)

	if endpoint == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, endpoint)
}

func (api *API) CreateSource(w http.ResponseWriter, r *http.Request) {
	var source entities.Source
	source.Init()
	defaults.Set(&source)
	if err := json.NewDecoder(r.Body).Decode(&source); err != nil {
		api.error(400, w, err)
		return
	}

	if err := source.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	source.WorkspaceId = ucontext.GetWorkspaceID(r.Context())
	err := api.DB.SourcesWS.Insert(r.Context(), &source)
	api.assert(err)

	api.json(201, w, source)
}

func (api *API) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	endpoint, err := api.DB.SourcesWS.Get(r.Context(), id)
	api.assert(err)
	if endpoint == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	if err := json.NewDecoder(r.Body).Decode(endpoint); err != nil {
		api.error(400, w, err)
		return
	}

	if err := endpoint.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	endpoint.ID = id
	err = api.DB.SourcesWS.Update(r.Context(), endpoint)
	api.assert(err)

	api.json(200, w, endpoint)
}

func (api *API) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	_, err := api.DB.SourcesWS.Delete(r.Context(), id)
	api.assert(err)

	w.WriteHeader(204)
}

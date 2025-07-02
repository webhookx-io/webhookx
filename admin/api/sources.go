package api

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"net/http"
)

func (api *API) PageSource(w http.ResponseWriter, r *http.Request) {
	var q query.SourceQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	list, total, err := api.db.SourcesWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetSource(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	source, err := api.db.SourcesWS.Get(r.Context(), id)
	api.assert(err)

	if source == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, source)
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
	err := api.db.SourcesWS.Insert(r.Context(), &source)
	api.assert(err)

	api.json(201, w, source)
}

func (api *API) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	source, err := api.db.SourcesWS.Get(r.Context(), id)
	api.assert(err)
	if source == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	if err := json.NewDecoder(r.Body).Decode(source); err != nil {
		api.error(400, w, err)
		return
	}

	if err := source.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	source.ID = id
	err = api.db.SourcesWS.Update(r.Context(), source)
	api.assert(err)

	api.json(200, w, source)
}

func (api *API) DeleteSource(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	_, err := api.db.SourcesWS.Delete(r.Context(), id)
	api.assert(err)

	w.WriteHeader(204)
}

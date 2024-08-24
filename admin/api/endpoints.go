package api

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"net/http"
)

func (api *API) PageEndpoint(w http.ResponseWriter, r *http.Request) {
	var q query.EndpointQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	list, total, err := api.DB.EndpointsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	endpoint, err := api.DB.EndpointsWS.Get(r.Context(), id)
	api.assert(err)

	if endpoint == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, endpoint)
}

func (api *API) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var endpoint entities.Endpoint
	endpoint.Init()
	defaults.Set(&endpoint)
	if err := json.NewDecoder(r.Body).Decode(&endpoint); err != nil {
		api.error(400, w, err)
		return
	}

	if err := endpoint.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	endpoint.WorkspaceId = ucontext.GetWorkspaceID(r.Context())
	err := api.DB.EndpointsWS.Insert(r.Context(), &endpoint)
	api.assert(err)

	api.json(201, w, endpoint)
}

func (api *API) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	endpoint, err := api.DB.EndpointsWS.Get(r.Context(), id)
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
	err = api.DB.EndpointsWS.Update(r.Context(), endpoint)
	api.assert(err)

	api.json(200, w, endpoint)
}

func (api *API) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	_, err := api.DB.EndpointsWS.Delete(r.Context(), id)
	api.assert(err)

	w.WriteHeader(204)
}

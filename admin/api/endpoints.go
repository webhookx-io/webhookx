package api

import (
	"net/http"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) PageEndpoint(w http.ResponseWriter, r *http.Request) {
	var q query.EndpointQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	list, total, err := api.db.EndpointsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	endpoint, err := api.db.EndpointsWS.Get(r.Context(), id)
	api.assert(err)

	if endpoint == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, endpoint)
}

func (api *API) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	var endpoint entities.Endpoint
	defaults := map[string]interface{}{"id": utils.KSUID()}
	if err := ValidateRequest(r, defaults, &endpoint); err != nil {
		api.error(400, w, err)
		return
	}

	endpoint.WorkspaceId = contextx.GetWorkspaceID(r.Context())
	err := api.db.EndpointsWS.Insert(r.Context(), &endpoint)
	api.assert(err)

	api.json(201, w, endpoint)
}

func (api *API) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	endpoint, err := api.db.EndpointsWS.Get(r.Context(), id)
	api.assert(err)
	if endpoint == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	defaults := utils.Must(utils.StructToMap(endpoint))
	if err := ValidateRequest(r, defaults, endpoint); err != nil {
		api.error(400, w, err)
		return
	}

	endpoint.ID = id
	err = api.db.EndpointsWS.Update(r.Context(), endpoint)
	api.assert(err)

	api.json(200, w, endpoint)
}

func (api *API) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	_, err := api.db.EndpointsWS.Delete(r.Context(), id)
	api.assert(err)

	w.WriteHeader(204)
}

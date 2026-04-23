package api

import (
	"net/http"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) PageSource(w http.ResponseWriter, r *http.Request) {
	parameters := api.lookupOperation("/workspaces/{ws_id}/sources", http.MethodGet).Parameters
	if err := openapi.ValidateParameters(r, parameters); err != nil {
		api.error(400, w, err)
		return
	}

	var params SourceListParams
	if err := api.bindQuery(r, &params); err != nil {
		api.error(400, w, err)
		return
	}

	query := params.Query()
	cursor, err := api.db.SourcesWS.Cursor(r.Context(), query)
	api.assert(err)

	api.json(200, w, BuildPaginationResponse(cursor, r.URL))
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
	defaults := map[string]interface{}{"id": utils.KSUID()}
	if err := ValidateRequest(r, defaults, &source); err != nil {
		api.error(400, w, err)
		return
	}

	if source.Type == "http" {
		if source.Config.HTTP.Path == "" {
			source.Config.HTTP.Path = "/" + utils.UUIDShort()
		}
	}

	source.WorkspaceId = contextx.GetWorkspaceID(r.Context())
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

	defaults := utils.Must(utils.StructToMap(source))
	if err := ValidateRequest(r, defaults, source); err != nil {
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

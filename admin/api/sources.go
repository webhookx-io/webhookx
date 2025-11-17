package api

import (
	"net/http"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/utils"
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

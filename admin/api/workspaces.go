package api

import (
	"errors"
	"net/http"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) PageWorkspace(w http.ResponseWriter, r *http.Request) {
	var q query.WorkspaceQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	list, total, err := api.db.Workspaces.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	workspace, err := api.db.Workspaces.Get(r.Context(), id)
	api.assert(err)

	if workspace == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, workspace)
}

func (api *API) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	if !license.GetLicenser().Allow("workspace") {
		api.json(403, w, types.ErrorResponse{Message: MsgLicenseInvalid})
		return
	}
	var workspace entities.Workspace
	defaults := map[string]interface{}{"id": utils.KSUID()}
	if err := ValidateRequest(r, defaults, &workspace); err != nil {
		api.error(400, w, err)
		return
	}

	workspace.ID = utils.KSUID()
	err := api.db.Workspaces.Insert(r.Context(), &workspace)
	api.assert(err)

	api.json(201, w, workspace)
}

func (api *API) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	workspace, err := api.db.Workspaces.Get(r.Context(), id)
	api.assert(err)
	if workspace == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	var name string
	if workspace.Name != nil {
		name = *workspace.Name
	}

	defaults := utils.Must(utils.StructToMap(workspace))
	if err := ValidateRequest(r, defaults, workspace); err != nil {
		api.error(400, w, err)
		return
	}

	if name == "default" && (workspace.Name == nil || *workspace.Name != "default") {
		api.error(400, w, errors.New("cannot rename default workspace"))
		return
	}

	workspace.ID = id
	err = api.db.Workspaces.Update(r.Context(), workspace)
	api.assert(err)

	api.json(200, w, workspace)
}

func (api *API) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	if !license.GetLicenser().Allow("workspace") {
		api.json(403, w, types.ErrorResponse{Message: MsgLicenseInvalid})
		return
	}
	id := api.param(r, "id")

	workspace, err := api.db.Workspaces.Get(r.Context(), id)
	api.assert(err)

	if workspace != nil {
		if workspace.Name != nil && *workspace.Name == "default" {
			api.error(400, w, errors.New("cannot delete a default workspace"))
			return
		}

		_, err = api.db.Workspaces.Delete(r.Context(), id)
		api.assert(err)
	}

	w.WriteHeader(204)
}

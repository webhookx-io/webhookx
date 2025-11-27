package api

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var workspace *entities.Workspace
		var err error

		wid := mux.Vars(r)["workspace"]
		if wid == "" {
			wid = "default"
		}

		workspace, err = api.db.Workspaces.GetWorkspace(r.Context(), wid)
		api.assert(err)
		if workspace == nil {
			workspace, err = api.db.Workspaces.Get(r.Context(), wid)
			api.assert(err)
		}

		if workspace == nil {
			api.error(400, w, errors.New("invalid workspace: "+wid))
			return
		}

		ctx := contextx.WithContext(r.Context(), &contextx.Context{
			WorkspaceID:   workspace.ID,
			WorkspaceName: utils.PointerValue(workspace.Name),
		})
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (api *API) licenseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := contextx.FromContext(r.Context())
		path, _ := mux.CurrentRoute(r).GetPathTemplate()
		method := r.Method
		if !license.GetLicenser().AllowAPI(ctx.WorkspaceName, path, method) {
			api.json(403, w, types.ErrorResponse{Message: MsgLicenseInvalid})
			return
		}
		next.ServeHTTP(w, r)
	})
}

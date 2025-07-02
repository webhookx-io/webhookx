package api

import (
	"errors"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"net/http"
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

		ctx := ucontext.WithContext(r.Context(), &ucontext.UContext{
			WorkspaceID: workspace.ID,
		})
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

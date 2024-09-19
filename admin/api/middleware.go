package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/errs"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

func (api *API) contextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var workspace *entities.Workspace
		var err error

		wid := mux.Vars(r)["workspace"]
		if wid == "" || wid == "default" {
			wid = "default"
			workspace, err = api.DB.Workspaces.GetDefault(r.Context())
		} else {
			workspace, err = api.DB.Workspaces.Get(r.Context(), wid)
		}
		api.assert(err)

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

func panicRecovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e := recover(); e != nil {
				var err error
				switch v := e.(type) {
				case error:
					err = v
				default:
					err = errors.New(fmt.Sprint(e))
				}

				if e, ok := err.(*errs.DBError); ok {
					w.Header().Set("Content-Type", ApplicationJsonType)
					w.WriteHeader(400)
					bytes, _ := json.Marshal(ErrorResponse{Message: e.Error()})
					_, _ = w.Write(bytes)
					return
				}

				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				buf = buf[:n]

				zap.S().Errorf("panic recovered: %v\n %s", err, buf)
				w.Header().Set("Content-Type", ApplicationJsonType)
				w.WriteHeader(500)
				_, _ = w.Write([]byte(`{"message": "internal error"}`))
			}
		}()

		h.ServeHTTP(w, r)
	})
}

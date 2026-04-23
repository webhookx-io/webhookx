package api

import (
	"net/http"

	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/pkg/types"
)

func (api *API) PageAttempt(w http.ResponseWriter, r *http.Request) {
	parameters := api.lookupOperation("/workspaces/{ws_id}/attempts", http.MethodGet).Parameters
	if err := openapi.ValidateParameters(r, parameters); err != nil {
		api.error(400, w, err)
		return
	}

	var params AttemptListParams
	if err := api.bindQuery(r, &params); err != nil {
		api.error(400, w, err)
		return
	}

	query := params.Query()
	cursor, err := api.db.AttemptsWS.Cursor(r.Context(), query)
	api.assert(err)

	api.json(200, w, BuildPaginationResponse(cursor, r.URL))
}

func (api *API) GetAttempt(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	attempt, err := api.db.AttemptsWS.Get(r.Context(), id)
	api.assert(err)

	if attempt == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	if attempt.AttemptedAt != nil {
		detail, err := api.db.AttemptDetailsWS.Get(r.Context(), attempt.ID)
		api.assert(err)
		if detail != nil {
			if detail.RequestHeaders != nil {
				attempt.Request.Headers = detail.RequestHeaders
			}
			if detail.RequestBody != nil {
				attempt.Request.Body = detail.RequestBody
			}
			if detail.ResponseHeaders != nil {
				attempt.Response.Headers = *detail.ResponseHeaders
			}
			if detail.ResponseBody != nil {
				attempt.Response.Body = detail.ResponseBody
			}
		}
	}

	api.json(200, w, attempt)
}

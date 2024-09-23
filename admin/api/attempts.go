package api

import (
	"net/http"

	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) PageAttempt(w http.ResponseWriter, r *http.Request) {
	var q query.AttemptQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)
	if r.URL.Query().Get("event_id") != "" {
		q.EventId = utils.Pointer(r.URL.Query().Get("event_id"))
	}
	if r.URL.Query().Get("endpoint_id") != "" {
		q.EndpointId = utils.Pointer(r.URL.Query().Get("endpoint_id"))
	}
	list, total, err := api.DB.AttemptsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetAttempt(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	attempt, err := api.DB.AttemptsWS.Get(r.Context(), id)
	api.assert(err)

	if attempt == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	if attempt.AttemptedAt != nil {
		attemptDetail, err := api.DB.AttemptDetailsWS.Get(r.Context(), attempt.ID)
		api.assert(err)
		attempt.Extend(attemptDetail)
	}

	api.json(200, w, attempt)
}

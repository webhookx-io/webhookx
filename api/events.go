package api

import (
	"context"
	"encoding/json"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
)

func (api *API) PageEvent(w http.ResponseWriter, r *http.Request) {
	var q query.EventQuery
	api.bindQuery(r, &q.Query)

	list, total, err := api.DB.Events.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	event, err := api.DB.Events.Get(r.Context(), id)
	api.assert(err)

	if event == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, event)
}

func (api *API) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event entities.Event
	event.ID = utils.UUID()

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		api.error(400, w, err)
		return
	}

	if err := event.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	err := HandleEvent(r.Context(), &event)
	api.assert(err)

	api.json(201, w, event)
}

func HandleEvent(ctx context.Context, event *entities.Event) error {
	return nil
}

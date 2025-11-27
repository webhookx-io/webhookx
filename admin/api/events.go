package api

import (
	"context"
	"net/http"
	"time"

	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/eventbus"
	"github.com/webhookx-io/webhookx/pkg/contextx"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

func (api *API) PageEvent(w http.ResponseWriter, r *http.Request) {
	var q query.EventQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)

	list, total, err := api.db.EventsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	event, err := api.db.EventsWS.Get(r.Context(), id)
	api.assert(err)

	if event == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, event)
}

func (api *API) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event entities.Event
	defaults := map[string]interface{}{"id": utils.KSUID()}
	if err := ValidateRequest(r, defaults, &event); err != nil {
		api.error(400, w, err)
		return
	}

	event.IngestedAt = types.Time{Time: time.Now()}
	event.WorkspaceId = contextx.GetWorkspaceID(r.Context())
	attempts, err := api.dispatcher.Dispatch(context.WithoutCancel(r.Context()), []*entities.Event{&event})
	api.assert(err)

	if len(attempts) > 0 {
		ids := make([]string, len(attempts))
		for i, attempt := range attempts {
			ids[i] = attempt.ID
		}
		err = api.bus.ClusteringBroadcast(eventbus.EventEventFanout, &eventbus.EventFanoutData{
			EventId:    event.ID,
			AttemptIds: ids,
		})
		api.assert(err)
	}

	dbEvent, err := api.db.EventsWS.Get(r.Context(), event.ID)
	api.assert(err)

	api.json(201, w, dbEvent)
}

func (api *API) RetryEvent(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	event, err := api.db.EventsWS.Get(r.Context(), id)
	api.assert(err)
	if event == nil {
		api.json(404, w, types.ErrorResponse{Message: MsgNotFound})
		return
	}

	endpointId := r.URL.Query().Get("endpoint_id")
	endpoint, err := api.db.EndpointsWS.Get(r.Context(), endpointId)
	api.assert(err)
	if endpoint == nil {
		api.json(400, w, types.ErrorResponse{Message: "endpoint not found"})
		return
	}

	attempts, err := api.dispatcher.DispatchEndpoint(r.Context(), event, []*entities.Endpoint{endpoint})
	api.assert(err)

	ids := make([]string, len(attempts))
	for i, attempt := range attempts {
		ids[i] = attempt.ID
	}
	err = api.bus.ClusteringBroadcast(eventbus.EventEventFanout, &eventbus.EventFanoutData{
		EventId:    event.ID,
		AttemptIds: ids,
	})
	api.assert(err)

	api.json(200, w, nil)
}

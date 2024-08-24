package api

import (
	"context"
	"encoding/json"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/model"
	"github.com/webhookx-io/webhookx/pkg/queue"
	"github.com/webhookx-io/webhookx/pkg/ucontext"
	"github.com/webhookx-io/webhookx/utils"
	"net/http"
)

func (api *API) PageEvent(w http.ResponseWriter, r *http.Request) {
	var q query.EventQuery
	q.Order("id", query.DESC)
	api.bindQuery(r, &q.Query)

	list, total, err := api.DB.EventsWS.Page(r.Context(), &q)
	api.assert(err)

	api.json(200, w, NewPagination(total, list))
}

func (api *API) GetEvent(w http.ResponseWriter, r *http.Request) {
	id := api.param(r, "id")
	event, err := api.DB.EventsWS.Get(r.Context(), id)
	api.assert(err)

	if event == nil {
		api.json(404, w, ErrorResponse{Message: MsgNotFound})
		return
	}

	api.json(200, w, event)
}

func (api *API) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event entities.Event
	event.ID = utils.KSUID()

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		api.error(400, w, err)
		return
	}

	if err := event.Validate(); err != nil {
		api.error(400, w, err)
		return
	}

	event.WorkspaceId = ucontext.GetWorkspaceID(r.Context())
	err := DispatchEvent(api, r.Context(), &event)
	api.assert(err)

	api.json(201, w, event)
}

func DispatchEvent(api *API, ctx context.Context, event *entities.Event) error {
	endpoints, err := listSubscribedEndpoints(ctx, api.DB, event.EventType)
	if err != nil {
		return err
	}

	attempts := make([]*entities.Attempt, 0, len(endpoints))
	tasks := make([]*queue.TaskMessage, 0, len(endpoints))

	err = api.DB.TX(ctx, func(ctx context.Context) error {
		err := api.DB.Events.Insert(ctx, event)
		if err != nil {
			return err
		}

		for _, endpoint := range endpoints {
			attempt := &entities.Attempt{
				ID:            utils.KSUID(),
				EventId:       event.ID,
				EndpointId:    endpoint.ID,
				Status:        entities.AttemptStatusInit,
				AttemptNumber: 1,
			}
			attempt.WorkspaceId = endpoint.WorkspaceId

			task := &queue.TaskMessage{
				ID: attempt.ID,
				Data: &model.MessageData{
					EventID:    event.ID,
					EndpointId: endpoint.ID,
					Delay:      endpoint.Retry.Config.Attempts[0],
					Attempt:    1,
				},
			}
			attempts = append(attempts, attempt)
			tasks = append(tasks, task)
		}

		return api.DB.AttemptsWS.BatchInsert(ctx, attempts)
	})
	if err != nil {
		return err
	}

	for _, task := range tasks {
		err := api.queue.Add(task, utils.DurationS(task.Data.(*model.MessageData).Delay))
		if err != nil {
			api.log.Warnf("failed to add task to queue: %v", err)
		}
		err = api.DB.AttemptsWS.UpdateStatus(ctx, task.ID, entities.AttemptStatusQueued)
		if err != nil {
			api.log.Warnf("failed to update attempt status: %v", err)
		}
	}

	return nil
}

func listSubscribedEndpoints(ctx context.Context, db *db.DB, eventType string) (list []*entities.Endpoint, err error) {
	var q query.EndpointQuery
	endpoints, err := db.EndpointsWS.List(ctx, &q)
	if err != nil {
		return nil, err
	}

	for _, endpoint := range endpoints {
		if !endpoint.Enabled {
			continue
		}
		for _, event := range endpoint.Events {
			if eventType == event {
				list = append(list, endpoint)
			}
		}
	}

	return
}

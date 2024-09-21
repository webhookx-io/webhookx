package admin

import (
	"context"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("/events", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application
	var db *db.DB
	var ws *entities.Workspace

	BeforeAll(func() {
		db = helper.InitDB(true, nil)
		app = utils.Must(helper.Start(map[string]string{
			"WEBHOOKX_ADMIN_LISTEN": "0.0.0.0:8080",
		}))
		ws = utils.Must(db.Workspaces.GetDefault(context.TODO()))
		adminClient = helper.AdminClient()
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("GET", func() {
		Context("with data", func() {

			var events []*entities.Event

			BeforeAll(func() {
				assert.NoError(GinkgoT(), db.Truncate("events"))
				for i := 1; i <= 21; i++ {
					event := &entities.Event{
						ID:        utils.KSUID(),
						EventType: "foo.bar",
						Data:      []byte("{}"),
					}
					event.WorkspaceId = ws.ID
					assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), event))
					events = append(events, event)
				}
			})

			It("retrieves first page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Event]{}).
					Get("/workspaces/default/events")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Event])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})

			It("retrieves second page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Event]{}).
					Get("/workspaces/default/events?page_no=2")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Event])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})
		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			var entity *entities.Event
			BeforeAll(func() {
				entitiesConfig := helper.EntitiesConfig{
					Events: []*entities.Event{
						{
							ID:        utils.KSUID(),
							EventType: "foo.bar",
							Data:      []byte("{}"),
						},
					},
				}
				entity = entitiesConfig.Events[0]
				helper.InitDB(false, &entitiesConfig)
			})

			It("retrieves by id", func() {
				resp, err := adminClient.R().
					SetResult(entities.Event{}).
					Get("/workspaces/default/events/" + entity.ID)

				assert.NoError(GinkgoT(), err)
				result := resp.Result().(*entities.Event)
				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), "foo.bar", result.EventType)
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Get("/workspaces/default/events/notfound")
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
			})
		})
	})

	Context("/{id}/retry", func() {
		Context("manually retry", func() {
			var endpointId, eventId string

			BeforeAll(func() {
				entitiesConfig := helper.EntitiesConfig{
					Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
					Events:    []*entities.Event{helper.DefaultEvent()},
				}
				endpointId = entitiesConfig.Endpoints[0].ID
				eventId = entitiesConfig.Events[0].ID

				helper.InitDB(false, &entitiesConfig)
			})

			It("should persist an attempt", func() {
				resp, err := adminClient.R().
					Post("/workspaces/default/events/" + eventId + "/retry?endpoint_id=" + endpointId)
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())

				q := query.AttemptQuery{
					EventId:    &eventId,
					EndpointId: &endpointId,
				}
				attempts, err := db.Attempts.List(context.TODO(), &q)
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 1, len(attempts))
				assert.Equal(GinkgoT(), entities.AttemptTriggerModeManual, attempts[0].TriggerMode)
				assert.Equal(GinkgoT(), entities.AttemptStatusQueued, attempts[0].Status)
			})

		})
	})

})

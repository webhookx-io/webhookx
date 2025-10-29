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
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"time"
)

var _ = Describe("/events", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application
	var db *db.DB
	var ws *entities.Workspace

	BeforeAll(func() {
		db = helper.InitDB(true, nil)
		app = utils.Must(helper.Start(map[string]string{}))
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
						ID:         utils.KSUID(),
						EventType:  "foo.bar",
						Data:       []byte("{}"),
						IngestedAt: types.Time{Time: time.Now()},
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

	Context("POST", func() {
		It("creates an event", func() {
			resp, err := adminClient.R().
				SetBody(`{
				    "event_type": "foo.bar",
				    "data": {"key":"value"}
				}`).
				SetResult(entities.Event{}).
				Post("/workspaces/default/events")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Event)
			assert.NotEmpty(GinkgoT(), result.ID)
			assert.Equal(GinkgoT(), "foo.bar", result.EventType)
			assert.Equal(GinkgoT(), `{"key":"value"}`, string(result.Data))
			assert.True(GinkgoT(), result.CreatedAt.Unix() > 0)
			assert.True(GinkgoT(), result.UpdatedAt.Unix() > 0)
		})

		Context("errors", func() {
			It("returns HTTP 400 for invalid json", func() {
				resp, err := adminClient.R().
					SetBody("").
					SetResult(entities.Event{}).
					Post("/workspaces/default/events")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})

			It("returns HTTP 400 for missing required fields", func() {
				resp, err := adminClient.R().
					SetBody(`{}`).
					SetResult(entities.Event{}).
					Post("/workspaces/default/events")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"data":"required field missing","event_type":"required field missing"}}}`,
					string(resp.Body()))
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
							ID:         utils.KSUID(),
							EventType:  "foo.bar",
							Data:       []byte("{}"),
							IngestedAt: types.Time{Time: time.Now()},
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
				assert.Equal(GinkgoT(), entity.IngestedAt.UnixMilli(), result.IngestedAt.UnixMilli())
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
					Endpoints: []*entities.Endpoint{factory.EndpointP()},
					Events:    []*entities.Event{factory.EventP()},
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
				assert.Equal(GinkgoT(), entities.AttemptStatusInit, attempts[0].Status)
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Post("/workspaces/default/events/notfound/retry")
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
				It("return HTTP 400", func() {
					resp, err := adminClient.R().Post("/workspaces/default/events/" + eventId + "/retry?endpoint_id=notfound")
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 400, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"endpoint not found\"}", string(resp.Body()))
				})
			})
		})

	})

})

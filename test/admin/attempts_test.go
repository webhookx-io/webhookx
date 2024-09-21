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
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"time"
)

var _ = Describe("/attempts", Ordered, func() {

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

			var endpoints []*entities.Endpoint
			var events []*entities.Event

			BeforeAll(func() {
				assert.NoError(GinkgoT(), db.Truncate("attempts"))
				endpoint1 := entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
				}
				endpoint1.WorkspaceId = ws.ID
				assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), &endpoint1))
				endpoints = append(endpoints, &endpoint1)

				endpoint2 := entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
				}
				endpoint2.WorkspaceId = ws.ID
				assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), &endpoint2))
				endpoints = append(endpoints, &endpoint2)

				for i := 1; i <= 21; i++ {
					event := &entities.Event{
						ID:        utils.KSUID(),
						EventType: "foo.bar",
						Data:      []byte("{}"),
					}
					event.WorkspaceId = ws.ID
					assert.NoError(GinkgoT(), db.Events.Insert(context.TODO(), event))
					events = append(events, event)
					attempt := entities.Attempt{
						ID:            utils.KSUID(),
						EventId:       event.ID,
						EndpointId:    endpoints[i%2].ID,
						Status:        entities.AttemptStatusSuccess,
						AttemptNumber: 1,
						ScheduledAt:   types.Time{Time: time.Now()},
						AttemptedAt:   &types.Time{Time: time.Now()},
					}
					attempt.WorkspaceId = ws.ID
					assert.NoError(GinkgoT(), db.Attempts.Insert(context.TODO(), &attempt))
				}
			})

			It("retrieves first page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Attempt]{}).
					Get("/workspaces/default/attempts")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Attempt])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})

			It("retrieves second page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Attempt]{}).
					Get("/workspaces/default/attempts?page_no=2")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Attempt])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})

			It("query by event_id", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Attempt]{}).
					Get("/workspaces/default/attempts?event_id=" + events[0].ID)
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Attempt])
				assert.EqualValues(GinkgoT(), 1, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})

			It("query by endpoint_id", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Attempt]{}).
					Get("/workspaces/default/attempts?endpoint_id=" + endpoints[0].ID)
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Attempt])
				assert.EqualValues(GinkgoT(), 10, result.Total)
				assert.EqualValues(GinkgoT(), 10, len(result.Data))
			})

		})

		Context("with no data", func() {
			BeforeAll(func() {
				assert.NoError(GinkgoT(), db.Truncate("attempts"))
			})
			It("retrieves first page", func() {
				resp, err := adminClient.R().Get("/workspaces/default/attempts")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), `{"total":0,"data":[]}`, string(resp.Body()))
			})
		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			var entity *entities.Attempt
			BeforeAll(func() {
				entitiesConfig := helper.EntitiesConfig{
					Endpoints: []*entities.Endpoint{
						{
							ID:      utils.KSUID(),
							Enabled: true,
							Request: entities.RequestConfig{
								URL:    "https://example.com",
								Method: "POST",
							},
						},
					},
					Events: []*entities.Event{
						{
							ID:        utils.KSUID(),
							EventType: "foo.bar",
							Data:      []byte("{}"),
						},
					},
				}
				entity = &entities.Attempt{
					ID:            utils.KSUID(),
					EventId:       entitiesConfig.Events[0].ID,
					EndpointId:    entitiesConfig.Endpoints[0].ID,
					Status:        entities.AttemptStatusSuccess,
					AttemptNumber: 1,
					ScheduledAt:   types.Time{Time: time.Now()},
					AttemptedAt:   &types.Time{Time: time.Now()},
				}
				entitiesConfig.Attempts = []*entities.Attempt{entity}

				helper.InitDB(false, &entitiesConfig)
			})

			It("retrieves by id", func() {
				resp, err := adminClient.R().
					SetResult(entities.Attempt{}).
					Get("/workspaces/default/attempts/" + entity.ID)

				assert.NoError(GinkgoT(), err)
				result := resp.Result().(*entities.Attempt)
				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), entity.EventId, result.EventId)
				assert.Equal(GinkgoT(), entity.EndpointId, result.EndpointId)
				assert.Equal(GinkgoT(), entities.AttemptStatusSuccess, result.Status)
				assert.EqualValues(GinkgoT(), 1, result.AttemptNumber)
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Get("/workspaces/default/attempts/notfound")
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
			})
		})
	})

})

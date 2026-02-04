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
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("/sources", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application
	var db *db.DB
	var ws *entities.Workspace

	BeforeAll(func() {
		db = helper.InitDB(true, nil)
		var err error
		adminClient = helper.AdminClient()
		app, err = helper.Start(nil)
		assert.Nil(GinkgoT(), err)
		ws, err = db.Workspaces.GetDefault(context.TODO())
		assert.Nil(GinkgoT(), err)
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("POST", func() {
		It("creates a source", func() {
			resp, err := adminClient.R().
				SetBody(`{ "type": "http", "config": { "http": { "path": "" } }}`).
				SetResult(entities.Source{}).
				Post("/workspaces/default/sources")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Source)
			assert.NotNil(GinkgoT(), result.ID)
			assert.Equal(GinkgoT(), true, result.Enabled)
			assert.True(GinkgoT(), len(result.Config.HTTP.Path) > 0) // auto generate path
			assert.EqualValues(GinkgoT(), []string{"POST"}, result.Config.HTTP.Methods)
			assert.Equal(GinkgoT(), false, result.Async)
			assert.True(GinkgoT(), nil == result.Config.HTTP.Response)

			e, err := db.Sources.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
		})

		Context("errors", func() {
			It("returns HTTP 400 for missing required fields", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{}).
					SetResult(entities.Source{}).
					Post("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":"required field missing"}}}`,
					string(resp.Body()))
			})
		})
	})

	Context("GET", func() {
		Context("with data", func() {
			BeforeAll(func() {
				assert.Nil(GinkgoT(), db.Truncate("sources"))
				for i := 1; i <= 21; i++ {
					entity := entities.Source{
						ID:      utils.KSUID(),
						Enabled: true,
					}
					entity.WorkspaceId = ws.ID
					assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), &entity))
				}
			})
			It("retrieves first page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})
			It("retrieves second page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources?page_no=2")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})
		})

		Context("with no data", func() {
			BeforeAll(func() {
				assert.Nil(GinkgoT(), db.Truncate("sources"))
			})
			It("retrieves first page", func() {
				resp, err := adminClient.R().Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), `{"total":0,"data":[]}`, string(resp.Body()))
			})
		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			var entity *entities.Source
			BeforeAll(func() {
				entity = &entities.Source{
					ID:      utils.KSUID(),
					Enabled: true,
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), entity))
			})

			It("retrieves by id", func() {
				resp, err := adminClient.R().
					SetResult(entities.Source{}).
					Get("/workspaces/default/sources/" + entity.ID)

				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*entities.Source)

				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), entity.Enabled, result.Enabled)
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Get("/workspaces/default/sources/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
			})
		})

		Context("PUT", func() {
			var entity *entities.Source

			BeforeAll(func() {
				entity = factory.SourceWS(ws.ID)
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), entity))
			})

			It("updates by id", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"type": "http",
						"config": map[string]interface{}{
							"http": map[string]interface{}{
								"path":    "/v1",
								"methods": []string{"GET", "POST", "PUT", "DELETE"},
								"response": map[string]interface{}{
									"code":         200,
									"content_type": "text/plain",
									"body":         "OK",
								},
							},
						},
						"async": true,
					}).
					SetResult(entities.Source{}).
					Put("/workspaces/default/sources/" + entity.ID)

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				result := resp.Result().(*entities.Source)

				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), "/v1", result.Config.HTTP.Path)
				assert.EqualValues(GinkgoT(), []string{"GET", "POST", "PUT", "DELETE"}, result.Config.HTTP.Methods)
				assert.Equal(GinkgoT(), true, result.Async)
				assert.EqualValues(GinkgoT(), &entities.CustomResponse{
					Code:        200,
					ContentType: "text/plain",
					Body:        "OK",
				}, result.Config.HTTP.Response)
			})
		})

		Context("DELETE", func() {
			var entity *entities.Source

			BeforeAll(func() {
				entity = factory.SourceWS(ws.ID)
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), entity))
			})

			It("deletes by id", func() {
				resp, err := adminClient.R().Delete("/workspaces/default/sources/" + entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 204, resp.StatusCode())
				e, err := db.Sources.Get(context.TODO(), entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Nil(GinkgoT(), e)
			})

			Context("errors", func() {
				It("return HTTP 204", func() {
					resp, err := adminClient.R().Delete("/workspaces/default/sources/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 204, resp.StatusCode())
				})
			})
		})
	})

})

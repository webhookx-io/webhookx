package admin

import (
	"context"
	"fmt"
	"strconv"

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
		BeforeAll(func() {
			assert.Nil(GinkgoT(), db.Truncate("sources"))
			for i := 1; i <= 21; i++ {
				entity := entities.Source{
					ID:      fmt.Sprintf("%03d", i),
					Name:    new(fmt.Sprintf("%03d", i)),
					Enabled: i%2 == 0,
					Metadata: map[string]string{
						"foo":   "bar",
						"value": strconv.Itoa(i),
					},
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), &entity))
			}
		})

		Context("offset pagination", func() {
			It("default", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})

			It("page_no", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Source]{}).
					SetQueryParam("page_no", "2").
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})

			It("page_size", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Source]{}).
					SetQueryParam("page_size", "21").
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 21, len(result.Data))
			})
		})

		Context("cursor pagination", func() {
			It("limit", func() {
				resp, err := adminClient.R().
					SetQueryParam("limit", "5").
					SetResult(api.CursorPagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.CursorPagination[*entities.Source])
				assert.Equal(GinkgoT(), 5, len(result.Data))
				assert.NotNil(GinkgoT(), result.Next)

				resp, err = adminClient.R().
					SetQueryParam("limit", "100").
					SetResult(api.CursorPagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.CursorPagination[*entities.Source])
				assert.Equal(GinkgoT(), 21, len(result.Data))
				assert.Nil(GinkgoT(), result.Next)
			})

			It("after", func() {
				// Get first 2
				resp, err := adminClient.R().
					SetQueryParam("limit", "2").
					SetQueryParam("sort", "id.asc").
					SetResult(api.CursorPagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.CursorPagination[*entities.Source])
				assert.Equal(GinkgoT(), 2, len(result.Data))
				firstId := result.Data[0].ID
				secondId := result.Data[1].ID

				// Get after first
				resp, err = adminClient.R().
					SetQueryParam("limit", "1").
					SetQueryParam("sort", "id.asc").
					SetQueryParam("after", firstId).
					SetResult(api.CursorPagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.CursorPagination[*entities.Source])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), secondId, result.Data[0].ID)
			})

			It("before", func() {
				// Get first 2
				resp, err := adminClient.R().
					SetQueryParam("limit", "2").
					SetQueryParam("sort", "id.asc").
					SetResult(api.CursorPagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.CursorPagination[*entities.Source])
				firstId := result.Data[0].ID
				secondId := result.Data[1].ID

				// Get before second
				resp, err = adminClient.R().
					SetQueryParam("limit", "1").
					SetQueryParam("sort", "id.asc").
					SetQueryParam("before", secondId).
					SetResult(api.CursorPagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.CursorPagination[*entities.Source])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), firstId, result.Data[0].ID)
			})

			Context("errors", func() {
				It("limit must be integer", func() {
					resp, err := adminClient.R().
						SetQueryParam("limit", "test").
						Get("/workspaces/default/sources")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"@params":["limit: value test: an invalid integer: invalid syntax"]}}}`,
						string(resp.Body()))

				})
				It("limit must in range [1, 1000]", func() {
					resp, err := adminClient.R().
						SetQueryParam("limit", "0").
						Get("/workspaces/default/sources")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"@params":["limit: number must be at least 1"]}}}`,
						string(resp.Body()))

					resp, err = adminClient.R().
						SetQueryParam("limit", "1001").
						Get("/workspaces/default/sources")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"@params":["limit: number must be at most 1000"]}}}`,
						string(resp.Body()))
				})
			})
		})

		Context("query parameters", func() {
			It("name", func() {
				resp, err := adminClient.R().
					SetQueryParam("name", "005").
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), "005", *result.Data[0].Name)
			})

			It("enabled", func() {
				resp, err := adminClient.R().
					SetQueryParam("enabled", "true").
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.True(GinkgoT(), len(result.Data) > 0)
				for _, ep := range result.Data {
					assert.True(GinkgoT(), ep.Enabled)
				}
			})

			It("created_at", func() {
				resp, err := adminClient.R().
					SetQueryParam("name", "005").
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				ep5 := resp.Result().(*api.Pagination[*entities.Source]).Data[0]

				resp, err = adminClient.R().
					SetQueryParam("created_at[lt]", fmt.Sprintf("%d", ep5.CreatedAt.UnixMilli()+1)).
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.True(GinkgoT(), len(result.Data) >= 1)
			})

			It("metadata", func() {
				resp, err := adminClient.R().
					SetQueryParam("metadata[value]", "5").
					SetResult(api.Pagination[*entities.Source]{}).
					Get("/workspaces/default/sources")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Source])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), "005", *result.Data[0].Name)
				assert.EqualValues(GinkgoT(), entities.Metadata{
					"foo": "bar",
					"value": "5",
				}, result.Data[0].Metadata)

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

package admin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("/endpoints", Ordered, func() {

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
		It("creates an endpoint", func() {
			now := time.Now()
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"request": map[string]interface{}{
						"url": "https://example.com",
					},
				}).
				SetResult(entities.Endpoint{}).
				Post("/workspaces/default/endpoints")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Endpoint)
			assert.NotNil(GinkgoT(), result.ID)
			assert.Equal(GinkgoT(), true, result.Enabled)

			assert.Equal(GinkgoT(), "https://example.com", result.Request.URL)
			assert.Equal(GinkgoT(), "POST", result.Request.Method)
			assert.EqualValues(GinkgoT(), 10000, result.Request.Timeout)

			assert.Equal(GinkgoT(), entities.RetryStrategyFixed, result.Retry.Strategy)
			assert.Equal(GinkgoT(), []int64{0, 60, 3600}, result.Retry.Config.Attempts)
			assert.Nil(GinkgoT(), result.RateLimit)

			e, err := db.Endpoints.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)

			assert.True(GinkgoT(), now.UnixMilli() <= e.UpdatedAt.UnixMilli())
			assert.True(GinkgoT(), now.UnixMilli() <= e.UpdatedAt.UnixMilli())
			assert.Equal(GinkgoT(), e.CreatedAt, e.UpdatedAt)
		})

		Context("errors", func() {
			It("returns HTTP 400 for invalid json", func() {
				resp, err := adminClient.R().
					SetBody("").
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})

			It("returns HTTP 400 for missing required fields", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{}).
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"request":{"url":"required field missing"}}}}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 for invalid request", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url": "",
						},
					}).
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"request":{"url":"minimum string length is 1"}}}}`,
					string(resp.Body()))
			})

			It("return HTTP 400 for unique constraint violation", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url":    "https://example.com",
							"method": "POST",
						},
						"name": "test",
					}).
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 201, resp.StatusCode())

				resp, err = adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url":    "https://example.com",
							"method": "POST",
						},
						"name": "test",
					}).
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				expected := fmt.Sprintf(`{"message":"unique constraint violation: {ws_id='%s',name='test'} already exists"}`, ws.ID)
				assert.Equal(GinkgoT(), expected, string(resp.Body()))

				resp2, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url":    "https://example.com",
							"method": "POST",
						},
						"name": "test2",
					}).
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 201, resp2.StatusCode())

				resp, err = adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url":    "https://example.com",
							"method": "POST",
						},
						"name": "test",
					}).
					SetResult(entities.Endpoint{}).
					Put("/workspaces/default/endpoints/" + resp2.Result().(*entities.Endpoint).ID)
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.JSONEq(GinkgoT(), expected, string(resp.Body()))
			})

			It("return HTTP 400 for invalid rate_limit: missing required properties", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url": "https://example.com",
						},
						"rate_limit": map[string]interface{}{},
					}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"Request Validation","error":{"message":"request validation","fields":{"rate_limit":{"period":"required field missing","quota":"required field missing"}}}}`, string(resp.Body()))
			})

			It("return HTTP 400 for invalid rate_limit: invalid properties", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url": "https://example.com",
						},
						"rate_limit": map[string]interface{}{
							"quota":  -1,
							"period": 1,
						},
					}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"Request Validation","error":{"message":"request validation","fields":{"rate_limit":{"quota":"number must be at least 0"}}}}`, string(resp.Body()))

				resp, err = adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url": "https://example.com",
						},
						"rate_limit": map[string]interface{}{
							"quota":  0,
							"period": 0,
						},
					}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"Request Validation","error":{"message":"request validation","fields":{"rate_limit":{"period":"number must be at least 1"}}}}`, string(resp.Body()))
			})
		})
	})

	Context("GET", func() {
		BeforeAll(func() {
			assert.Nil(GinkgoT(), db.Truncate("endpoints"))
			for i := 1; i <= 21; i++ {
				entity := entities.Endpoint{
					ID:      fmt.Sprintf("ep_%03d", i),
					Name:    utils.Pointer(fmt.Sprintf("endpoint_%03d", i)),
					Enabled: i%2 == 0,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
					Metadata: map[string]string{
						"foo":   "bar",
						"value": strconv.Itoa(i),
					},
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), &entity))
			}
		})

		Context("offset pagination", func() {
			It("default", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})

			It("page_no", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Endpoint]{}).
					SetQueryParam("page_no", "2").
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})

			It("page_size", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Endpoint]{}).
					SetQueryParam("page_size", "21").
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 21, len(result.Data))
			})
		})

		Context("cursor pagination", func() {
			It("limit", func() {
				resp, err := adminClient.R().
					SetQueryParam("limit", "5").
					SetResult(api.PaginationCursor[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.PaginationCursor[*entities.Endpoint])
				assert.Equal(GinkgoT(), 5, len(result.Data))
				assert.NotNil(GinkgoT(), result.Next)

				resp, err = adminClient.R().
					SetQueryParam("limit", "100").
					SetResult(api.PaginationCursor[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.PaginationCursor[*entities.Endpoint])
				assert.Equal(GinkgoT(), 21, len(result.Data))
				assert.Nil(GinkgoT(), result.Next)
			})

			It("after", func() {
				// Get first 2
				resp, err := adminClient.R().
					SetQueryParam("limit", "2").
					SetQueryParam("sort", "id.asc").
					SetResult(api.PaginationCursor[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.PaginationCursor[*entities.Endpoint])
				assert.Equal(GinkgoT(), 2, len(result.Data))
				firstId := result.Data[0].ID
				secondId := result.Data[1].ID

				// Get after first
				resp, err = adminClient.R().
					SetQueryParam("limit", "1").
					SetQueryParam("sort", "id.asc").
					SetQueryParam("after", firstId).
					SetResult(api.PaginationCursor[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.PaginationCursor[*entities.Endpoint])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), secondId, result.Data[0].ID)
			})

			It("before", func() {
				// Get first 2
				resp, err := adminClient.R().
					SetQueryParam("limit", "2").
					SetQueryParam("sort", "id.asc").
					SetResult(api.PaginationCursor[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.PaginationCursor[*entities.Endpoint])
				firstId := result.Data[0].ID
				secondId := result.Data[1].ID

				// Get before second
				resp, err = adminClient.R().
					SetQueryParam("limit", "1").
					SetQueryParam("sort", "id.asc").
					SetQueryParam("before", secondId).
					SetResult(api.PaginationCursor[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.PaginationCursor[*entities.Endpoint])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), firstId, result.Data[0].ID)
			})

			Context("errors", func() {
				It("limit must be integer", func() {
					resp, err := adminClient.R().
						SetQueryParam("limit", "test").
						Get("/workspaces/default/endpoints")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"@params":["limit: value test: an invalid integer: invalid syntax"]}}}`,
						string(resp.Body()))

				})
				It("limit must in range [1, 1000]", func() {
					resp, err := adminClient.R().
						SetQueryParam("limit", "0").
						Get("/workspaces/default/endpoints")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"@params":["limit: number must be at least 1"]}}}`,
						string(resp.Body()))

					resp, err = adminClient.R().
						SetQueryParam("limit", "1001").
						Get("/workspaces/default/endpoints")
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
					SetQueryParam("name", "endpoint_005").
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), "endpoint_005", *result.Data[0].Name)
			})

			It("enabled", func() {
				resp, err := adminClient.R().
					SetQueryParam("enabled", "true").
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.True(GinkgoT(), len(result.Data) > 0)
				for _, ep := range result.Data {
					assert.True(GinkgoT(), ep.Enabled)
				}
			})

			It("created_at", func() {
				resp, err := adminClient.R().
					SetQueryParam("name", "endpoint_005").
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				ep5 := resp.Result().(*api.Pagination[*entities.Endpoint]).Data[0]

				resp, err = adminClient.R().
					SetQueryParam("created_at[lt]", fmt.Sprintf("%d", ep5.CreatedAt.UnixMilli()+1)).
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.True(GinkgoT(), len(result.Data) >= 1)

				resp, err = adminClient.R().
					SetQueryParam("created_at[gte]", fmt.Sprintf("%d", ep5.CreatedAt.UnixMilli())).
					SetQueryParam("created_at[lte]", fmt.Sprintf("%d", ep5.CreatedAt.UnixMilli())).
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result = resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.True(GinkgoT(), len(result.Data) >= 1)
			})

			It("metadata", func() {
				resp, err := adminClient.R().
					SetQueryParam("metadata[value]", "5").
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.Equal(GinkgoT(), 1, len(result.Data))
				assert.Equal(GinkgoT(), "endpoint_005", *result.Data[0].Name)
				assert.EqualValues(GinkgoT(), entities.Metadata{
					"foo": "bar",
					"value": "5",
				}, result.Data[0].Metadata)

			})
		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			var entity *entities.Endpoint
			BeforeAll(func() {
				entity = &entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), entity))
			})

			It("retrieves by id", func() {
				resp, err := adminClient.R().
					SetResult(entities.Endpoint{}).
					Get("/workspaces/default/endpoints/" + entity.ID)

				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*entities.Endpoint)

				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), entity.Enabled, result.Enabled)
				assert.Equal(GinkgoT(), entity.Request, result.Request)
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Get("/workspaces/default/endpoints/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
			})
		})

		Context("PUT", func() {
			var entity *entities.Endpoint
			BeforeAll(func() {
				entity = &entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
					Events: []string{},
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), entity))
			})

			It("updates by id", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url":    "https://foo.com",
							"method": "PUT",
						},
						"retry": map[string]interface{}{
							"strategy": "fixed",
							"config": map[string]interface{}{
								"attempts": []int64{0, 30, 60},
							},
						},
					}).
					SetResult(entities.Endpoint{}).
					Put("/workspaces/default/endpoints/" + entity.ID)

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				result := resp.Result().(*entities.Endpoint)

				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), "https://foo.com", result.Request.URL)
				assert.Equal(GinkgoT(), "PUT", result.Request.Method)
			})
		})

		Context("DELETE", func() {
			var entity *entities.Endpoint
			BeforeAll(func() {
				entity = &entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), entity))
			})

			It("deletes by id", func() {
				resp, err := adminClient.R().Delete("/workspaces/default/endpoints/" + entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 204, resp.StatusCode())
				e, err := db.Endpoints.Get(context.TODO(), entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Nil(GinkgoT(), e)
			})

			Context("errors", func() {
				It("return HTTP 204", func() {
					resp, err := adminClient.R().Delete("/workspaces/default/endpoints/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 204, resp.StatusCode())
				})
			})
		})
	})

})

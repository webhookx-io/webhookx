package admin

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
)

var _ = Describe("/endpoints", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application
	var db *db.DB
	var ws *entities.Workspace

	BeforeAll(func() {
		assert.Nil(GinkgoT(), helper.ResetDB())
		db = helper.DB()
		var err error
		adminClient = helper.AdminClient()
		app, err = test.Start(map[string]string{
			"WEBHOOKX_ADMIN_LISTEN": "0.0.0.0:8080",
		})
		assert.Nil(GinkgoT(), err)
		ws, err = db.Workspaces.GetDefault(context.TODO())
		assert.Nil(GinkgoT(), err)
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("POST", func() {
		It("creates an endpoint", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"request": map[string]interface{}{
						"url":    "https://example.com",
						"method": "POST",
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

			e, err := db.Endpoints.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
		})

		Context("errors", func() {
			It("returns HTTP 400 for invalid json", func() {
				resp, err := adminClient.R().
					SetBody("").
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				fmt.Println(string(resp.Body()))
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
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"method":"invalid method","url":"invalid url"}}}`,
					string(resp.Body()))
			})
		})
	})

	Context("GET", func() {
		Context("with data", func() {
			BeforeAll(func() {
				assert.Nil(GinkgoT(), db.Truncate("endpoints"))
				for i := 1; i <= 21; i++ {
					entity := entities.Endpoint{
						ID:      utils.KSUID(),
						Enabled: true,
						Request: entities.RequestConfig{
							URL:    "https://example.com",
							Method: "POST",
						},
					}
					entity.WorkspaceId = ws.ID
					assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), &entity))
				}
			})
			It("retrieves first page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})
			It("retrieves second page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Endpoint]{}).
					Get("/workspaces/default/endpoints?page_no=2")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Endpoint])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})
		})

		Context("with no data", func() {
			BeforeAll(func() {
				assert.Nil(GinkgoT(), db.Truncate("endpoints"))
			})
			It("retrieves first page", func() {
				resp, err := adminClient.R().Get("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), `{"total":0,"data":[]}`, string(resp.Body()))
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

func TestEndpointsAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Endpoints Suite")
}

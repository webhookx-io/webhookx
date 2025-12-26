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
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("/workspaces", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application
	var db *db.DB
	var ws *entities.Workspace

	BeforeAll(func() {
		db = helper.InitDB(true, nil)
		var err error
		adminClient = helper.AdminClient()
		app = helper.MustStart(
			map[string]string{},
			helper.WithLicenser(&helper.MockLicenser{}))
		ws, err = db.Workspaces.GetDefault(context.TODO())
		assert.Nil(GinkgoT(), err)
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("POST", func() {
		It("creates a workspace", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"name": "foo",
				}).
				SetResult(entities.Workspace{}).
				Post("/workspaces")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Workspace)
			assert.NotNil(GinkgoT(), result.ID)
			assert.Equal(GinkgoT(), "foo", *result.Name)

			e, err := db.Workspaces.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
		})

		Context("errors", func() {
			It("returns HTTP 400 for invalid json", func() {
				resp, err := adminClient.R().
					SetBody("").
					Post("/workspaces")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})

			It("return HTTP 400 for unique constraint violation", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name": "default",
					}).
					Post("/workspaces")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				expected := `{"message": "unique constraint violation: {name='default'} already exists"}`
				assert.JSONEq(GinkgoT(), expected, string(resp.Body()))
			})
		})
	})

	Context("GET", func() {
		Context("with data", func() {
			It("retrieves first page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Workspace]{}).
					Get("/workspaces")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Workspace])
				assert.True(GinkgoT(), result.Total > 0)
			})
		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			It("retrieves by id", func() {
				resp, err := adminClient.R().
					SetResult(entities.Workspace{}).
					Get("/workspaces/" + ws.ID)

				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*entities.Workspace)

				assert.Equal(GinkgoT(), ws.ID, result.ID)
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Get("/workspaces/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
			})
		})

		Context("PUT", func() {
			var entity *entities.Workspace
			BeforeAll(func() {
				entity = &entities.Workspace{
					ID:   utils.KSUID(),
					Name: utils.Pointer("test-update-workspace"),
				}
				assert.Nil(GinkgoT(), db.Workspaces.Insert(context.TODO(), entity))
			})

			It("updates by id", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"description": utils.Pointer("test"),
					}).
					SetResult(entities.Workspace{}).
					Put("/workspaces/" + entity.ID)

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				result := resp.Result().(*entities.Workspace)

				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), "test-update-workspace", *result.Name)
				assert.Equal(GinkgoT(), "test", *result.Description)
			})

			It("cannot rename default workspace", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name": utils.Pointer("other"),
					}).
					SetResult(entities.Workspace{}).
					Put("/workspaces/" + ws.ID)
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(), "{\"message\":\"cannot rename default workspace\"}", string(resp.Body()))
			})
		})

		Context("DELETE", func() {
			var entity *entities.Workspace
			BeforeAll(func() {
				entity = &entities.Workspace{
					ID: utils.KSUID(),
				}
				assert.Nil(GinkgoT(), db.Workspaces.Insert(context.TODO(), entity))
			})

			It("deletes by id", func() {
				resp, err := adminClient.R().Delete("/workspaces/" + entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 204, resp.StatusCode())
				e, err := db.Workspaces.Get(context.TODO(), entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Nil(GinkgoT(), e)
			})

			Context("errors", func() {
				It("return HTTP 204", func() {
					resp, err := adminClient.R().Delete("/workspaces/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 204, resp.StatusCode())
				})
				It("cannot delete default workspace", func() {
					resp, err := adminClient.R().Delete("/workspaces/" + ws.ID)
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 400, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"cannot delete a default workspace\"}", string(resp.Body()))
				})
			})
		})
	})

})

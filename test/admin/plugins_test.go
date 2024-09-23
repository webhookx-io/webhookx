package admin

import (
	"context"
	"encoding/json"
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

var _ = Describe("/plugins", Ordered, func() {

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
			BeforeAll(func() {
				assert.NoError(GinkgoT(), db.Truncate("endpoints"))
				assert.NoError(GinkgoT(), db.Truncate("plugins"))
				for i := 1; i <= 21; i++ {
					endpoint := helper.DefaultEndpoint()
					endpoint.WorkspaceId = ws.ID
					assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))
					plugin := entities.Plugin{
						ID:         utils.KSUID(),
						EndpointId: endpoint.ID,
						Name:       "webhookx-signature",
						Enabled:    true,
						Config:     json.RawMessage("{}"),
					}
					plugin.WorkspaceId = ws.ID
					assert.NoError(GinkgoT(), db.Plugins.Insert(context.TODO(), &plugin))
				}
			})

			It("retrieves first page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Plugin]{}).
					Get("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Plugin])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 20, len(result.Data))
			})

			It("retrieves second page", func() {
				resp, err := adminClient.R().
					SetResult(api.Pagination[*entities.Plugin]{}).
					Get("/workspaces/default/plugins?page_no=2")
				assert.Nil(GinkgoT(), err)
				result := resp.Result().(*api.Pagination[*entities.Plugin])
				assert.EqualValues(GinkgoT(), 21, result.Total)
				assert.EqualValues(GinkgoT(), 1, len(result.Data))
			})

		})

		Context("with no data", func() {
			BeforeAll(func() {
				assert.NoError(GinkgoT(), db.Truncate("plugins"))
			})
			It("retrieves first page", func() {
				resp, err := adminClient.R().Get("/workspaces/default/plugins")
				assert.NoError(GinkgoT(), err)
				assert.Equal(GinkgoT(), `{"total":0,"data":[]}`, string(resp.Body()))
			})
		})
	})

	Context("POST", func() {
		var endpoint *entities.Endpoint

		BeforeEach(func() {
			endpoint = helper.DefaultEndpoint()
			assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))
		})

		It("creates a plugin", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"name":        "webhookx-signature",
					"endpoint_id": endpoint.ID,
				}).
				SetResult(entities.Plugin{}).
				Post("/workspaces/default/plugins")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Plugin)
			assert.NotNil(GinkgoT(), result.ID)
			assert.Equal(GinkgoT(), endpoint.ID, result.EndpointId)
			assert.Equal(GinkgoT(), "webhookx-signature", result.Name)
			assert.Equal(GinkgoT(), true, result.Enabled)
			data := make(map[string]string)
			json.Unmarshal(result.Config, &data)
			assert.Equal(GinkgoT(), 32, len(data["key"]))

			e, err := db.Plugins.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
		})

		It("creates a plugin with plugin config", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"name":        "webhookx-signature",
					"endpoint_id": endpoint.ID,
					"config": map[string]string{
						"key": "abcde",
					},
				}).
				SetResult(entities.Plugin{}).
				Post("/workspaces/default/plugins")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Plugin)
			assert.NotNil(GinkgoT(), result.ID)
			assert.Equal(GinkgoT(), endpoint.ID, result.EndpointId)
			assert.Equal(GinkgoT(), "webhookx-signature", result.Name)
			assert.Equal(GinkgoT(), true, result.Enabled)
			data := make(map[string]string)
			json.Unmarshal(result.Config, &data)
			assert.Equal(GinkgoT(), "abcde", data["key"])

			e, err := db.Plugins.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			var entity *entities.Plugin
			BeforeAll(func() {
				entitiesConfig := helper.EntitiesConfig{
					Endpoints: []*entities.Endpoint{helper.DefaultEndpoint()},
				}
				entity = &entities.Plugin{
					ID:         utils.KSUID(),
					EndpointId: entitiesConfig.Endpoints[0].ID,
					Name:       "webhookx-signature",
					Enabled:    true,
					Config:     json.RawMessage(`{"key":"abcde"}`),
				}
				entitiesConfig.Plugins = []*entities.Plugin{entity}

				helper.InitDB(false, &entitiesConfig)
			})

			It("retrieves by id", func() {
				resp, err := adminClient.R().
					SetResult(entities.Plugin{}).
					Get("/workspaces/default/plugins/" + entity.ID)

				assert.NoError(GinkgoT(), err)
				result := resp.Result().(*entities.Plugin)
				assert.Equal(GinkgoT(), entity.ID, result.ID)
				assert.Equal(GinkgoT(), entity.EndpointId, result.EndpointId)
				assert.Equal(GinkgoT(), entity.Name, result.Name)
				assert.Equal(GinkgoT(), entity.Enabled, result.Enabled)
				assert.Equal(GinkgoT(), `{"key": "abcde"}`, string(entity.Config))
			})

			Context("errors", func() {
				It("return HTTP 404", func() {
					resp, err := adminClient.R().Get("/workspaces/default/plugins/notfound")
					assert.NoError(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), "{\"message\":\"Not found\"}", string(resp.Body()))
				})
			})
		})
	})

})

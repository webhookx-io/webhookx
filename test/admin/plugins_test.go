package admin

import (
	"context"
	"strings"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	key_auth "github.com/webhookx-io/webhookx/plugins/key-auth"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/hello"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/inbound"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/outbound"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("/plugins", Ordered, func() {

	plugin.RegisterPlugin(plugin.TypeInbound, "inbound", func() plugin.Plugin {
		return &inbound.InboundPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeOutbound, "outbound", func() plugin.Plugin {
		return &outbound.OutboundPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeOutbound, "hello", func() plugin.Plugin {
		return &hello.HelloPlugin{}
	})

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
			BeforeAll(func() {
				assert.NoError(GinkgoT(), db.Truncate("endpoints"))
				assert.NoError(GinkgoT(), db.Truncate("plugins"))
				for i := 1; i <= 21; i++ {
					endpoint := factory.EndpointWS(ws.ID)
					assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), &endpoint))
					plugin := entities.Plugin{
						ID:         utils.KSUID(),
						EndpointId: utils.Pointer(endpoint.ID),
						Name:       "webhookx-signature",
						Enabled:    true,
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

		Context("webhookx-signature plugin", func() {
			It("creates plugin with missing config", func() {
				endpoint := factory.EndpointP()
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))
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
				assert.Equal(GinkgoT(), endpoint.ID, *result.EndpointId)
				assert.Equal(GinkgoT(), "webhookx-signature", result.Name)
				assert.Equal(GinkgoT(), true, result.Enabled)
				assert.Equal(GinkgoT(), 32, len(result.Config["signing_secret"].(string)))

				e, err := db.Plugins.Get(context.TODO(), result.ID)
				assert.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), e)
			})

			It("creates plugin with plugin config", func() {
				endpoint := factory.EndpointP()
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":        "webhookx-signature",
						"endpoint_id": endpoint.ID,
						"config": map[string]string{
							"signing_secret": "abcde",
						},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)

				assert.Equal(GinkgoT(), 201, resp.StatusCode())

				result := resp.Result().(*entities.Plugin)
				assert.NotNil(GinkgoT(), result.ID)
				assert.Equal(GinkgoT(), endpoint.ID, *result.EndpointId)
				assert.Equal(GinkgoT(), "webhookx-signature", result.Name)
				assert.Equal(GinkgoT(), true, result.Enabled)
				assert.Equal(GinkgoT(), "abcde", result.Config["signing_secret"].(string))

				e, err := db.Plugins.Get(context.TODO(), result.ID)
				assert.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), e)
			})

			It("creates plugin with unknown properties", func() {
				endpoint := factory.EndpointP()
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":        "webhookx-signature",
						"endpoint_id": endpoint.ID,
						"config": map[string]string{
							"signing_secret": "abcde",
							"unknown":        "unknown",
						},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)

				assert.Equal(GinkgoT(), 201, resp.StatusCode())

				result := resp.Result().(*entities.Plugin)
				assert.NotNil(GinkgoT(), result.ID)
				assert.Equal(GinkgoT(), endpoint.ID, *result.EndpointId)
				assert.Equal(GinkgoT(), "webhookx-signature", result.Name)
				assert.Equal(GinkgoT(), true, result.Enabled)
				// the unknown property should be removed
				assert.EqualValues(GinkgoT(), map[string]interface{}{"signing_secret": "abcde"}, result.Config)

				e, err := db.Plugins.Get(context.TODO(), result.ID)
				assert.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), e)
			})
		})

		Context("function plugin", func() {
			It("return 400 when function exceed the maximum length", func() {
				source := factory.SourceP()
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), source))
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":      "function",
						"source_id": source.ID,
						"config": map[string]string{
							"function": strings.Repeat("a", 1048576+1),
						},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"function":"maximum string length is 1048576"}}}}`,
					string(resp.Body()))
			})
		})

		Context("basic-auth plugin", func() {
			It("return 201", func() {
				source := factory.SourceP()
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), source))
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":      "basic-auth",
						"source_id": source.ID,
						"config": map[string]string{
							"username": "foo",
							"password": "bar",
						},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 201, resp.StatusCode())

				result := resp.Result().(*entities.Plugin)
				assert.Equal(GinkgoT(), "basic-auth", result.Name)
				assert.Equal(GinkgoT(), source.ID, *result.SourceId)
				assert.Equal(GinkgoT(), "foo", result.Config["username"])
				assert.Equal(GinkgoT(), "bar", result.Config["password"])
			})
		})

		Context("key-auth plugin", func() {
			It("return 201", func() {
				source := factory.SourceP()
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), source))
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":      "key-auth",
						"source_id": source.ID,
						"config": map[string]interface{}{
							"param_name":      "apikey",
							"param_locations": []string{"header", "query"},
							"key":             "mykey",
						},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 201, resp.StatusCode())

				result := resp.Result().(*entities.Plugin)
				assert.Equal(GinkgoT(), "key-auth", result.Name)
				assert.Equal(GinkgoT(), source.ID, *result.SourceId)
				cfg := key_auth.Config{}
				utils.MapToStruct(result.Config, &cfg)
				assert.Equal(GinkgoT(), "apikey", cfg.ParamName)
				assert.Equal(GinkgoT(), []string{"header", "query"}, cfg.ParamLocations)
				assert.Equal(GinkgoT(), "mykey", cfg.Key)
			})
		})

		Context("hmac-auth plugin", func() {
			It("return 201", func() {
				source := factory.SourceP()
				assert.Nil(GinkgoT(), db.Sources.Insert(context.TODO(), source))
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":      "hmac-auth",
						"source_id": source.ID,
						"config": map[string]interface{}{
							"hash":             "sha-256",
							"encoding":         "base64",
							"signature_header": "x-signature",
							"secret":           "mykey",
						},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 201, resp.StatusCode())

				result := resp.Result().(*entities.Plugin)
				assert.Equal(GinkgoT(), "hmac-auth", result.Name)
				assert.Equal(GinkgoT(), source.ID, *result.SourceId)
				assert.Equal(GinkgoT(), "sha-256", result.Config["hash"])
				assert.Equal(GinkgoT(), "base64", result.Config["encoding"])
				assert.Equal(GinkgoT(), "x-signature", result.Config["signature_header"])
				assert.Equal(GinkgoT(), "mykey", result.Config["secret"])
			})
		})

		Context("errors", func() {
			It("return HTTP 400", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{"config": "string"}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":"value must be an object"}}}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 for unkown plugin name", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{"name": "unknown"}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"name":"unknown plugin name 'unknown'"}}}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 when missing endpoint_id for outbound type plugin", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{"name": "outbound"}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"endpoint_id":"endpoint_id is required for plugin 'outbound'"}}}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 when missing source_id for inbound type plugin", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{"name": "inbound"}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"source_id":"source_id is required for plugin 'inbound'"}}}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 when missing required config fields", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":        "hello",
						"endpoint_id": "test",
						"config":      map[string]interface{}{},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"message":"required field missing"}}}}`,
					string(resp.Body()))
			})

			It("return HTTP 400 when configuration filed type does not match", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":        "hello",
						"endpoint_id": "test",
						"config":      map[string]interface{}{"message": 1},
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"message":"value must be a string"}}}}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 when source_id does not exist", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":      "inbound",
						"source_id": "foo",
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"foreign key violation: {source_id='foo                        '} does not reference an existing record in 'sources'"}`,
					string(resp.Body()))
			})

			It("returns HTTP 400 when endpoint_id does not exist", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"name":        "outbound",
						"endpoint_id": "foo",
					}).
					SetResult(entities.Plugin{}).
					Post("/workspaces/default/plugins")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"foreign key violation: {endpoint_id='foo                        '} does not reference an existing record in 'endpoints'"}`,
					string(resp.Body()))
			})

		})
	})

	Context("/{id}", func() {
		Context("GET", func() {
			var entity *entities.Plugin
			BeforeAll(func() {
				entitiesConfig := helper.EntitiesConfig{
					Endpoints: []*entities.Endpoint{factory.EndpointP()},
				}
				entity = &entities.Plugin{
					ID:         utils.KSUID(),
					EndpointId: utils.Pointer(entitiesConfig.Endpoints[0].ID),
					Name:       "webhookx-signature",
					Enabled:    true,
					Config:     map[string]interface{}{"signing_secret": "abcde"},
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
				assert.EqualValues(GinkgoT(), map[string]interface{}{"signing_secret": "abcde"}, entity.Config)
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

		Context("PUT", func() {
			var endpoint *entities.Endpoint
			var plugin *entities.Plugin

			BeforeAll(func() {
				endpoint = &entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
				}
				endpoint.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

				plugin = &entities.Plugin{
					ID:         utils.KSUID(),
					Name:       "webhookx-signature",
					Enabled:    true,
					EndpointId: utils.Pointer(endpoint.ID),
				}
				plugin.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Plugins.Insert(context.TODO(), plugin))
			})

			It("updates by id", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"config": map[string]interface{}{
							"signing_secret": "foo",
						},
						"enabled": false,
					}).
					SetResult(entities.Plugin{}).
					Put("/workspaces/default/plugins/" + plugin.ID)

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				result := resp.Result().(*entities.Plugin)
				assert.Equal(GinkgoT(), plugin.ID, result.ID)
				assert.Equal(GinkgoT(), endpoint.ID, *result.EndpointId)
				assert.Equal(GinkgoT(), "webhookx-signature", result.Name)
				assert.Equal(GinkgoT(), false, result.Enabled)
			})

			Context("errors", func() {
				It("return HTTP 400 when configuration filed type does not match", func() {
					resp, err := adminClient.R().
						SetBody(map[string]interface{}{
							"config": map[string]interface{}{
								"signing_secret": 1,
							},
						}).
						SetResult(entities.Plugin{}).
						Put("/workspaces/default/plugins/" + plugin.ID)
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 400, resp.StatusCode())
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"signing_secret":"value must be a string"}}}}`,
						string(resp.Body()))
				})

				It("should return HTTP 400 for invalid request body", func() {
					resp, err := adminClient.R().
						SetBody("{ invalid json }").
						SetResult(entities.Plugin{}).
						Put("/workspaces/default/plugins/" + plugin.ID)
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 400, resp.StatusCode())
				})

				It("should return HTTP 404", func() {
					resp, err := adminClient.R().
						SetBody(map[string]interface{}{}).
						SetResult(entities.Plugin{}).
						Put("/workspaces/default/plugins/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 404, resp.StatusCode())
					assert.Equal(GinkgoT(), `{"message":"Not found"}`, string(resp.Body()))
				})
			})

		})

		Context("DELETE", func() {
			var entity *entities.Plugin
			BeforeAll(func() {
				endpoint := &entities.Endpoint{
					ID:      utils.KSUID(),
					Enabled: true,
					Request: entities.RequestConfig{
						URL:    "https://example.com",
						Method: "POST",
					},
				}
				endpoint.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

				entity = &entities.Plugin{
					ID:         utils.KSUID(),
					Name:       "webhookx-signature",
					Enabled:    true,
					EndpointId: utils.Pointer(endpoint.ID),
				}
				entity.WorkspaceId = ws.ID
				assert.Nil(GinkgoT(), db.Plugins.Insert(context.TODO(), entity))
			})

			It("deletes by id", func() {
				resp, err := adminClient.R().Delete("/workspaces/default/plugins/" + entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 204, resp.StatusCode())
				e, err := db.Plugins.Get(context.TODO(), entity.ID)
				assert.Nil(GinkgoT(), err)
				assert.Nil(GinkgoT(), e)
			})

			Context("errors", func() {
				It("return HTTP 204", func() {
					resp, err := adminClient.R().Delete("/workspaces/default/plugins/notfound")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 204, resp.StatusCode())
				})
			})
		})
	})

})

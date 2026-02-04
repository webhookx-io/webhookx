package admin

import (
	"context"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
)

var _ = Describe("metadata", Ordered, func() {

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
		ws, err = helper.GetDeafultWorkspace()
		assert.Nil(GinkgoT(), err)
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("endpoints", func() {
		It("retrieve a endpoint with empty metadata", func() {
			endpoint := factory.EndpointWS(ws.ID)
			assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

			resp, err := adminClient.R().
				SetResult(entities.Endpoint{}).
				Get("/workspaces/default/endpoints/" + endpoint.ID)
			assert.Nil(GinkgoT(), err)
			result := resp.Result().(*entities.Endpoint)
			assert.EqualValues(GinkgoT(), map[string]string{}, result.Metadata)
		})

		It("retrieve a endpoint with non-empty metadata", func() {
			endpoint := factory.Endpoint(func(o *entities.Endpoint) {
				o.Metadata = map[string]string{"k1": "v1", "k2": "v2"}
				o.WorkspaceId = ws.ID
			})
			assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

			resp, err := adminClient.R().
				SetResult(entities.Endpoint{}).
				Get("/workspaces/default/endpoints/" + endpoint.ID)
			assert.Nil(GinkgoT(), err)
			result := resp.Result().(*entities.Endpoint)
			assert.EqualValues(GinkgoT(), map[string]string{"k1": "v1", "k2": "v2"}, result.Metadata)
		})

		It("creates a endpoint with metadata", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"request": map[string]interface{}{
						"url":    "https://example.com",
						"method": "POST",
					},
					"metadata": map[string]string{
						"key": "value",
					},
				}).
				SetResult(entities.Endpoint{}).
				Post("/workspaces/default/endpoints")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			result := resp.Result().(*entities.Endpoint)
			assert.NotNil(GinkgoT(), result.ID)
			assert.EqualValues(GinkgoT(), map[string]string{"key": "value"}, result.Metadata)

			e, err := db.Endpoints.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
			assert.EqualValues(GinkgoT(), map[string]string{"key": "value"}, e.Metadata)
		})

		It("updates a endpoint's metadata", func() {
			endpoint := factory.Endpoint()
			endpoint.WorkspaceId = ws.ID
			assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"metadata": map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				}).
				SetResult(entities.Endpoint{}).
				Put("/workspaces/default/endpoints/" + endpoint.ID)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			result := resp.Result().(*entities.Endpoint)
			assert.EqualValues(GinkgoT(), map[string]string{"key1": "value1", "key2": "value2"}, result.Metadata)

			e, err := db.Endpoints.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
			assert.EqualValues(GinkgoT(), map[string]string{"key1": "value1", "key2": "value2"}, e.Metadata)
		})

		It("updates a endpoint's metadata that has existing keys", func() {
			endpoint := factory.EndpointWS(ws.ID, func(o *entities.Endpoint) {
				o.Metadata = map[string]string{"key1": "value1", "key2": "value2"}
			})
			assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

			// override metadata
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"metadata": map[string]string{
						"key3": "value3",
					},
				}).
				SetResult(entities.Endpoint{}).
				Put("/workspaces/default/endpoints/" + endpoint.ID)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			result := resp.Result().(*entities.Endpoint)
			assert.EqualValues(GinkgoT(), map[string]string{"key3": "value3"}, result.Metadata)

			e, err := db.Endpoints.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
			assert.EqualValues(GinkgoT(), map[string]string{"key3": "value3"}, e.Metadata)
		})

		It("updates a endpoint's metadata to be empty", func() {
			endpoint := factory.EndpointWS(ws.ID, func(o *entities.Endpoint) {
				o.Metadata = map[string]string{"key1": "value1", "key2": "value2"}
			})
			assert.Nil(GinkgoT(), db.Endpoints.Insert(context.TODO(), endpoint))

			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"metadata": map[string]string{},
				}).
				SetResult(entities.Endpoint{}).
				Put("/workspaces/default/endpoints/" + endpoint.ID)
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			result := resp.Result().(*entities.Endpoint)
			assert.EqualValues(GinkgoT(), map[string]string{}, result.Metadata)

			e, err := db.Endpoints.Get(context.TODO(), result.ID)
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), e)
			assert.EqualValues(GinkgoT(), map[string]string{}, e.Metadata)
		})

		Context("errors", func() {
			It("returns HTTP 400 for invalid metadata", func() {
				resp, err := adminClient.R().
					SetBody(map[string]interface{}{
						"request": map[string]interface{}{
							"url":    "https://example.com",
							"method": "POST",
						},
						"metadata": map[string]any{
							"k": 1,
						},
					}).
					SetResult(entities.Endpoint{}).
					Post("/workspaces/default/endpoints")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"metadata":{"k":"value must be a string"}}}}`,
					string(resp.Body()))
			})
		})
	})
})

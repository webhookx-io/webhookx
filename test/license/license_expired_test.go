package license

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

func setupWorkspaceEntities(db *db.DB, wid string) error {
	err := db.Endpoints.Insert(context.TODO(), factory.Endpoint(func(o *entities.Endpoint) {
		o.WorkspaceId = wid
	}))
	if err != nil {
		return err
	}
	return nil
}

var _ = Describe("expired", Ordered, func() {
	Context("admin", func() {
		var app *app.Application
		var adminClient *resty.Client

		BeforeAll(func() {
			db := helper.InitDB(true, nil)
			adminClient = helper.AdminClient()
			app = utils.Must(helper.Start(nil))

			// update license to be expired
			license.GetLicenser().License().ExpiredAt = time.Time{}

			defaultWorkspace, err := helper.GetDeafultWorkspace()
			assert.Nil(GinkgoT(), err)
			testWorkspace := factory.Workspace("test")
			assert.NoError(GinkgoT(), db.Workspaces.Insert(context.TODO(), testWorkspace))
			// setup default workspace
			assert.NoError(GinkgoT(), setupWorkspaceEntities(db, defaultWorkspace.ID))
			// setup test workspace
			assert.NoError(GinkgoT(), setupWorkspaceEntities(db, testWorkspace.ID))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("workspace creation should return 403", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"name": "foo",
				}).
				SetResult(entities.Workspace{}).
				Post("/workspaces")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 403, resp.StatusCode())
			assert.Equal(GinkgoT(), "{\"message\":\"license missing or expired\"}", string(resp.Body()))
		})

		It("workspace deletion should return 403", func() {
			resp, err := adminClient.R().Delete("/workspaces/default")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 403, resp.StatusCode())
			assert.Equal(GinkgoT(), "{\"message\":\"license missing or expired\"}", string(resp.Body()))
		})

		It("allow reading from default workspace", func() {
			resp, err := adminClient.R().
				SetResult(api.Pagination[*entities.Endpoint]{}).
				Get("/workspaces/default/endpoints")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			result := resp.Result().(*api.Pagination[*entities.Endpoint])
			assert.EqualValues(GinkgoT(), 1, result.Total)
			assert.EqualValues(GinkgoT(), 1, len(result.Data))

			resp, err = adminClient.R().
				SetResult(api.Pagination[*entities.Endpoint]{}).
				Get("/sources")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("allow writing to default workspace", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"request": map[string]interface{}{
						"url": "https://example.com",
					},
				}).
				Post("/workspaces/default/endpoints")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 201, resp.StatusCode())

			resp, err = adminClient.R().
				SetBody(`{ "type": "http", "config": { "http": { "path": "" } }}`).
				SetResult(entities.Source{}).
				Post("/sources")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 201, resp.StatusCode())
		})

		It("allow reading from different workspace", func() {
			resp, err := adminClient.R().
				SetResult(api.Pagination[*entities.Endpoint]{}).
				Get("/workspaces/test/endpoints")
			assert.Nil(GinkgoT(), err)
			result := resp.Result().(*api.Pagination[*entities.Endpoint])
			assert.EqualValues(GinkgoT(), 1, result.Total)
			assert.EqualValues(GinkgoT(), 1, len(result.Data))
		})

		It("deny writing to different workspace", func() {
			resp, err := adminClient.R().
				SetBody(map[string]interface{}{
					"request": map[string]interface{}{
						"url": "https://example.com",
					},
				}).
				Post("/workspaces/test/endpoints")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 403, resp.StatusCode())
			assert.Equal(GinkgoT(), "{\"message\":\"license missing or expired\"}", string(resp.Body()))
		})

	})
})

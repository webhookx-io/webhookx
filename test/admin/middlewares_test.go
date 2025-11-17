package admin

import (
	"context"
	"errors"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/admin/api"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

type ReturnErrWorkspaceDao struct {
	*dao.DAO[entities.Workspace]
}

func (dao *ReturnErrWorkspaceDao) Get(ctx context.Context, id string) (*entities.Workspace, error) {
	return nil, nil
}

func (dao *ReturnErrWorkspaceDao) GetDefault(ctx context.Context) (*entities.Workspace, error) {
	return nil, nil
}

func (dao *ReturnErrWorkspaceDao) GetWorkspace(ctx context.Context, name string) (*entities.Workspace, error) {
	return nil, errors.New("failed to get workspace")
}

var _ = Describe("middlewares", Ordered, func() {

	Context("panic middleware", func() {
		var app *app.Application
		var adminClient *resty.Client

		BeforeAll(func() {
			helper.InitDB(true, nil)
			adminClient = helper.AdminClient()
			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("return HTTP 500 when panic recovered", func() {
			app.DB().Workspaces = &ReturnErrWorkspaceDao{}
			resp, err := adminClient.R().Get("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 500, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"internal error"}`, string(resp.Body()))
		})
	})

	Context("context middleware", func() {
		var app *app.Application
		var db *db.DB
		var adminClient *resty.Client
		var testWorkspace *entities.Workspace

		BeforeAll(func() {
			db = helper.InitDB(true, nil)

			// test workspace
			testWorkspace = factory.Workspace("test")
			assert.NoError(GinkgoT(), db.Workspaces.Insert(context.TODO(), testWorkspace))
			testEndpoint := factory.EndpointWS(testWorkspace.ID)
			assert.NoError(GinkgoT(), db.Endpoints.Insert(context.TODO(), &testEndpoint))

			adminClient = helper.AdminClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("allows workspace name as url parameter", func() {
			resp, err := adminClient.R().
				SetResult(api.Pagination[*entities.Endpoint]{}).
				Get("/workspaces/test/endpoints")
			assert.Nil(GinkgoT(), err)
			result := resp.Result().(*api.Pagination[*entities.Endpoint])
			assert.EqualValues(GinkgoT(), 1, result.Total)
		})

		It("allows workspace id as url parameter", func() {
			resp, err := adminClient.R().
				SetResult(api.Pagination[*entities.Endpoint]{}).
				Get("/workspaces/" + testWorkspace.ID + "/endpoints")
			assert.Nil(GinkgoT(), err)
			result := resp.Result().(*api.Pagination[*entities.Endpoint])
			assert.EqualValues(GinkgoT(), 1, result.Total)
		})

		It("return 400 when workspace is not found", func() {
			resp, err := adminClient.R().Get("/workspaces/notfound/endpoints")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 400, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"invalid workspace: notfound"}`, string(resp.Body()))

			resp, err = adminClient.R().Get("/workspaces/2sw5MaDfC17ZzGqfewJKMf7Ow15/endpoints")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 400, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"invalid workspace: 2sw5MaDfC17ZzGqfewJKMf7Ow15"}`, string(resp.Body()))
		})
	})
})

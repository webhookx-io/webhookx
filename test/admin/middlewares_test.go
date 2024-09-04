package admin

import (
	"context"
	"errors"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/dao"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

type ReturnErrWorkspaceDao struct {
	*dao.DAO[entities.Workspace]
}

func (dao *ReturnErrWorkspaceDao) Get(ctx context.Context, id string) (*entities.Workspace, error) {
	return nil, nil
}

func (dao *ReturnErrWorkspaceDao) GetDefault(ctx context.Context) (*entities.Workspace, error) {
	return nil, errors.New("failed to get default workspace")
}

var _ = Describe("middlewares", Ordered, func() {
	var app *app.Application
	var adminClient *resty.Client

	BeforeAll(func() {
		app = utils.Must(helper.Start(map[string]string{
			"WEBHOOKX_ADMIN_LISTEN": "0.0.0.0:8080",
		}))
		app.DB().Workspaces = &ReturnErrWorkspaceDao{}
		adminClient = helper.AdminClient()
	})

	AfterAll(func() {
		app.Stop()
	})

	It("return HTTP 500 when panic recovered", func() {
		resp, err := adminClient.R().Get("/")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 500, resp.StatusCode())
		assert.Equal(GinkgoT(), `{"message": "internal error"}`, string(resp.Body()))
	})

	It("return HTTP 400 when workspace not found", func() {
		resp, err := adminClient.R().Get("/workspaces/notfound/endpoints")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 400, resp.StatusCode())
		assert.Equal(GinkgoT(), `{"message":"invalid workspace: notfound"}`, string(resp.Body()))
	})
})

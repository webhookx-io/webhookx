package license

import (
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("admin API", Ordered, func() {

	Context("/workspaces", func() {
		var app *app.Application
		var adminClient *resty.Client

		BeforeAll(func() {
			adminClient = helper.AdminClient()
			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			if app != nil {
				app.Stop()
			}
		})

		It("workspace creation should return 403 when no license", func() {
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

		It("workspace deletion should return 403 when no license", func() {
			resp, err := adminClient.R().Delete("/workspaces/default")
			assert.Nil(GinkgoT(), err)

			assert.Equal(GinkgoT(), 403, resp.StatusCode())
			assert.Equal(GinkgoT(), "{\"message\":\"license missing or expired\"}", string(resp.Body()))
		})

	})

})

package admin

import (
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("admin", Ordered, func() {

	Context("listen", func() {
		var app *app.Application
		var adminClient *resty.Client

		BeforeAll(func() {
			helper.InitDB(true, nil)
			app = utils.Must(helper.Start(map[string]string{}))
			adminClient = helper.AdminClient()
		})

		AfterAll(func() {
			app.Stop()
		})

		It("admin listen", func() {
			resp, err := adminClient.R().Get("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		It("404", func() {
			resp, err := adminClient.R().Get("/notfound")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 404, resp.StatusCode())
			assert.Equal(GinkgoT(), `{"message":"not found"}`, string(resp.Body()))
		})
	})

	Context("tls listen", func() {
		var app *app.Application
		var adminClient *resty.Client

		BeforeAll(func() {
			helper.InitDB(true, nil)
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_ADMIN_TLS_CERT": test.FilePath("server.crt"),
				"WEBHOOKX_ADMIN_TLS_KEY":  test.FilePath("server.key"),
			}))
			adminClient = helper.AdminTLSClient()
		})

		AfterAll(func() {
			app.Stop()
		})

		It("admin tls listen", func() {
			resp, err := adminClient.R().Get("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})
	})

})

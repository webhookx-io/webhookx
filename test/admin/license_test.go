package admin

import (
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
)

var _ = Describe("/license", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application

	BeforeAll(func() {
		adminClient = helper.AdminClient()
		app = helper.MustStart(map[string]string{})
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("GET", func() {
		It("retrieve license", func() {
			expected := `{
			    "id": "00000000-0000-0000-0000-000000000000",
			    "plan": "free",
			    "customer": "anonymous",
			    "expired_at": "2099-12-31T23:59:59Z",
			    "created_at": "1996-08-24T00:00:00Z",
			    "version": "1",
			    "signature": ""
			}`
			resp, err := adminClient.R().
				Get("/license")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.JSONEq(GinkgoT(), expected, string(resp.Body()))
		})
	})

})

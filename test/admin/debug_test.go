package admin

import (
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"time"
)

var _ = Describe("/debug", Ordered, func() {

	var adminClient *resty.Client
	var app *app.Application

	BeforeAll(func() {
		app = utils.Must(helper.Start(map[string]string{
			"WEBHOOKX_ADMIN_LISTEN":          "0.0.0.0:8080",
			"WEBHOOKX_ADMIN_DEBUG_ENDPOINTS": "true",
		}))
		adminClient = helper.AdminClient()
	})

	AfterAll(func() {
		app.Stop()
	})

	It("/debug/pprof/allocs", func() {
		assert.Eventually(GinkgoT(), func() bool {
			resp, err := adminClient.R().Get("/debug/pprof/allocs?debug=1")
			return err == nil && resp.StatusCode() == 200
		}, time.Second*10, time.Second)
	})

	It("/debug/pprof/block", func() {
		resp, err := adminClient.R().Get("/debug/pprof/block?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/goroutine", func() {
		resp, err := adminClient.R().Get("/debug/pprof/goroutine?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/heap", func() {
		resp, err := adminClient.R().Get("/debug/pprof/heap?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/mutex", func() {
		resp, err := adminClient.R().Get("/debug/pprof/mutex?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/threadcreate", func() {
		resp, err := adminClient.R().Get("/debug/pprof/threadcreate?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/cmdline", func() {
		resp, err := adminClient.R().Get("/debug/pprof/cmdline?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/symbol", func() {
		resp, err := adminClient.R().Get("/debug/pprof/symbol?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/profile", func() {
		resp, err := adminClient.R().Get("/debug/pprof/profile?seconds=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/trace", func() {
		resp, err := adminClient.R().Get("/debug/pprof/trace?seconds=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

})

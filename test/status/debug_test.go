package status

import (
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("/debug", Ordered, func() {

	var statusClient *resty.Client
	var app *app.Application

	BeforeAll(func() {
		helper.InitDB(true, nil)
		app = utils.Must(helper.Start(nil))
		statusClient = helper.StatusClient()

		assert.Eventually(GinkgoT(), func() bool {
			resp, err := statusClient.R().Get("/")
			return err == nil && resp.StatusCode() == 200
		}, time.Second*10, time.Second)
	})

	AfterAll(func() {
		app.Stop()
	})

	It("/debug/pprof/allocs", func() {
		resp, err := statusClient.R().Get("/debug/pprof/allocs?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/block", func() {
		resp, err := statusClient.R().Get("/debug/pprof/block?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/goroutine", func() {
		resp, err := statusClient.R().Get("/debug/pprof/goroutine?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/heap", func() {
		resp, err := statusClient.R().Get("/debug/pprof/heap?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/mutex", func() {
		resp, err := statusClient.R().Get("/debug/pprof/mutex?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/threadcreate", func() {
		resp, err := statusClient.R().Get("/debug/pprof/threadcreate?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/cmdline", func() {
		resp, err := statusClient.R().Get("/debug/pprof/cmdline?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/symbol", func() {
		resp, err := statusClient.R().Get("/debug/pprof/symbol?debug=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/profile", func() {
		resp, err := statusClient.R().Get("/debug/pprof/profile?seconds=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

	It("/debug/pprof/trace", func() {
		resp, err := statusClient.R().Get("/debug/pprof/trace?seconds=1")
		assert.NoError(GinkgoT(), err)
		assert.Equal(GinkgoT(), 200, resp.StatusCode())
	})

})

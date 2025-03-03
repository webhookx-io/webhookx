package admin

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
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

	It("/debug/pprof/", func() {
		paths := []string{
			"/debug/pprof/allocs",
			"/debug/pprof/block",
			"/debug/pprof/goroutine",
			"/debug/pprof/heap",
			"/debug/pprof/mutex",
			"/debug/pprof/threadcreate",
		}

		for _, path := range paths {
			resp, err := adminClient.R().Get(path + "?debug=1")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode(), fmt.Sprintf("%s \n%s", path, resp.Status()))
		}
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

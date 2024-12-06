package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/cmd"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("admin", Ordered, func() {
	Context("sync", func() {
		var app *app.Application

		BeforeAll(func() {
			helper.InitDB(true, nil)
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
				"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
				"WEBHOOKX_WORKER_ENABLED": "true",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			output, err := executeCommand(cmd.Command(), "admin", "sync", "../fixtures/webhookx.yaml")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), "", output)
		})
		Context("errors", func() {
			It("missing filename", func() {
				output, err := executeCommand(cmd.Command(), "admin", "sync")
				assert.NotNil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "Error: accepts 1 arg(s), received 0\n", output)
			})
			It("invalid filename", func() {
				output, err := executeCommand(cmd.Command(), "admin", "sync", "unknown.yaml")
				assert.NotNil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "Error: open unknown.yaml: no such file or directory\n", output)
			})
			It("timeout", func() {
				output, err := executeCommand(cmd.Command(), "admin", "sync", "../fixtures/webhookx.yaml", "--timeout", "0")
				assert.NotNil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "Error: timeout\n", output)
			})

		})
	})
})

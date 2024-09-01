package log

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"go.uber.org/zap"
	"testing"
)

var _ = Describe("logging", Ordered, func() {

	Context("TEXT", func() {
		var app *app.Application

		BeforeAll(func() {
			var err error
			app, err = test.Start(map[string]string{
				"WEBHOOKX_LOG_LEVEL":  "debug",
				"WEBHOOKX_LOG_FORMAT": "text",
				"WEBHOOKX_LOG_FILE":   "webhookx.log",
			})
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		BeforeEach(func() {
			helper.TruncateFile("webhookx.log")
		})

		Context("DEBUG", func() {
			It("outputs log with text format", func() {
				zap.S().Debugf("a debug log")
				line, err := helper.FileLine("webhookx.log", 1)
				assert.Nil(GinkgoT(), err)
				assert.Regexp(GinkgoT(), "^.+DEBUG.+a debug log$", line)
			})
		})
	})

	Context("JSON", func() {
		var app *app.Application
		BeforeAll(func() {
			var err error
			app, err = test.Start(map[string]string{
				"WEBHOOKX_LOG_LEVEL":  "debug",
				"WEBHOOKX_LOG_FORMAT": "json",
				"WEBHOOKX_LOG_FILE":   "webhookx.log",
			})
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		BeforeEach(func() {
			helper.TruncateFile("webhookx.log")
		})

		Context("DEBUG", func() {
			It("outputs log with json format", func() {
				zap.S().Debugf("a debug log")
				line, err := helper.FileLine("webhookx.log", 1)
				assert.Nil(GinkgoT(), err)
				data := make(map[string]interface{})
				assert.Nil(GinkgoT(), json.Unmarshal([]byte(line), &data))
				assert.Equal(GinkgoT(), "a debug log", data["msg"])
				assert.Equal(GinkgoT(), "debug", data["level"])
			})
		})
	})

})

func TestLogging(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logging Suite")
}

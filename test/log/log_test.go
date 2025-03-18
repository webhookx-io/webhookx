package log

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
	"testing"
	"time"
)

var _ = Describe("logging", Ordered, func() {

	Context("TEXT", func() {
		var app *app.Application

		BeforeAll(func() {
			var err error
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_LOG_LEVEL":  "debug",
				"WEBHOOKX_LOG_FORMAT": "text",
				"WEBHOOKX_LOG_FILE":   test.FilePath("output/webhookx-text.log"),
			}))
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		Context("DEBUG", func() {
			It("outputs log with text format", func() {
				var n = 0
				var err error
				assert.Eventually(GinkgoT(), func() bool {
					zap.S().Sync()
					n, err = helper.FileCountLine(test.FilePath("output/webhookx-text.log"))
					assert.Nil(GinkgoT(), err)
					return err == nil
				}, time.Second*5, time.Second)
				zap.S().Debugf("a debug log")
				zap.S().Sync()
				line, err := helper.FileLine(test.FilePath("output/webhookx-text.log"), n+1)
				assert.Nil(GinkgoT(), err)
				assert.Regexp(GinkgoT(), "^.+DEBUG.+a debug log$", line)
			})
		})
	})

	Context("JSON", func() {
		var app *app.Application
		BeforeAll(func() {
			var err error
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_LOG_LEVEL":  "debug",
				"WEBHOOKX_LOG_FORMAT": "json",
				"WEBHOOKX_LOG_FILE":   test.FilePath("output/webhookx-json.log"),
			}))
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		Context("DEBUG", func() {
			It("outputs log with json format", func() {
				var n = 0
				var err error
				assert.Eventually(GinkgoT(), func() bool {
					zap.S().Sync()
					n, err = helper.FileCountLine(test.FilePath("output/webhookx-json.log"))
					return err == nil
				}, time.Second*5, time.Second)
				zap.S().Debugf("a debug log")
				zap.S().Sync()
				line, err := helper.FileLine(test.FilePath("output/webhookx-json.log"), n+1)
				assert.Nil(GinkgoT(), err)
				data := make(map[string]interface{})
				assert.Nil(GinkgoT(), json.Unmarshal([]byte(line), &data), line)
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

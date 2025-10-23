package admin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
)

var _ = Describe("anonymous reports", Ordered, func() {

	Context("anonymous_reports = false", func() {
		var app *app.Application

		BeforeAll(func() {
			helper.InitDB(true, nil)
			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_ANONYMOUS_REPORTS": "false",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should display log when anonymous_reports is disabled", func() {
			matched, err := helper.FileHasLine(helper.LogFile, "^.*anonymous reports is disabled$")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})
	})

})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AnonymousReport Suite")
}

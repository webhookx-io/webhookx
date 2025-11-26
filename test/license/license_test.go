package license

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/test"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("License", Ordered, func() {

	Context("load", func() {
		It("should succeed when using a valid license", func() {
			licenseJSON, err := os.ReadFile(test.FilePath("fixtures/license.json"))
			app, err := helper.Start(map[string]string{
				"WEBHOOKX_LICENSE": string(licenseJSON),
			})
			assert.Nil(GinkgoT(), err)
			assert.NotNil(GinkgoT(), app)
			app.Stop()
		})

		Context("errors", func() {
			It("malformed license", func() {
				_, err := helper.Start(map[string]string{
					"WEBHOOKX_LICENSE": "{⚠️}",
				})
				assert.EqualError(GinkgoT(), err, "license is invalid: failed to parse license: invalid character 'â' looking for beginning of object key string")
			})

			It("invalid license", func() {
				_, err := helper.Start(map[string]string{
					"WEBHOOKX_LICENSE": "{}",
				})
				assert.EqualError(GinkgoT(), err, "license is invalid: signature is invalid")
			})
		})
	})

	Context("license state", func() {
		var app *app.Application

		BeforeAll(func() {
			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			if app != nil {
				app.Stop()
			}
		})

		It("should log when license expired", func() {
			license.GetLicenser().License().ExpiredAt = time.Time{}
			app.Scheduler().GetTask("license.expiration").Do()
			matched, err := helper.FileHasLine(helper.LogFile, "^.*license expired$")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})

		It("should log when license expiration less than 30 days", func() {
			license.GetLicenser().License().ExpiredAt = time.Now().AddDate(0, 0, 1)
			app.Scheduler().GetTask("license.expiration").Do()
			matched, err := helper.FileHasLine(helper.LogFile, fmt.Sprintf("^.*license will expire at.*$"))
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})

		It("should log when license expiration less than 90 days", func() {
			license.GetLicenser().License().ExpiredAt = time.Now().UTC().AddDate(0, 0, 31)
			app.Scheduler().GetTask("license.expiration").Do()
			matched, err := helper.FileHasLine(helper.LogFile, fmt.Sprintf("^.*license will expire on \\d{4}-\\d{2}-\\d{2}$"))
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})
	})

})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "License Suite")
}

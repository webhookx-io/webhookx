package admin

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/config"
	"github.com/webhookx-io/webhookx/pkg/reports"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("anonymous reports", Ordered, func() {

	defaultURL := reports.URL
	BeforeAll(func() {
		reports.URL = "http://localhost:8888"
	})

	AfterAll(func() {
		reports.URL = defaultURL
	})

	Context("reports", func() {
		Context("free", func() {
			var app *app.Application

			BeforeAll(func() {
				app = utils.Must(helper.Start(map[string]string{}))
			})

			AfterAll(func() {
				app.Stop()
			})

			It("report anonymous data", func() {
				var data map[string]interface{}
				server := helper.StartHttpServer(func(w http.ResponseWriter, r *http.Request) {
					err := json.NewDecoder(r.Body).Decode(&data)
					if err != nil {
						panic(err)
					}
				}, ":8888")

				task := app.Scheduler().GetTask("anonymous_reports")
				task.Do()

				assert.Equal(GinkgoT(), "free", data["license_plan"])
				assert.Equal(GinkgoT(), "00000000-0000-0000-0000-000000000000", data["license_id"])
				assert.Equal(GinkgoT(), config.VERSION, data["version"])
				server.Close()
			})
		})
	})

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
			assert.Eventually(GinkgoT(), func() bool {
				matched, err := helper.FileHasLine(helper.LogFile, "^.*anonymous reports is disabled$")
				assert.Nil(GinkgoT(), err)
				return matched
			}, time.Second, time.Millisecond*100)
		})
	})

})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AnonymousReport Suite")
}

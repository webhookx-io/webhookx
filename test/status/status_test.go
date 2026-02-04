package status

import (
	"errors"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/status"
	"github.com/webhookx-io/webhookx/status/health"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("status", Ordered, func() {

	Context("/", func() {
		var app *app.Application
		var statusClient *resty.Client

		BeforeAll(func() {
			helper.InitDB(true, nil)
			app = utils.Must(helper.Start(nil))
			statusClient = helper.StatusClient()
		})

		AfterAll(func() {
			app.Stop()
		})

		It("/", func() {
			resp, err := statusClient.R().Get("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})
	})

	Context("/health", func() {
		Context("UP", func() {
			var app *app.Application
			var statusClient *resty.Client

			BeforeAll(func() {
				helper.InitDB(true, nil)
				app = utils.Must(helper.Start(nil))
				statusClient = helper.StatusClient()
			})

			AfterAll(func() {
				app.Stop()
			})
			It("returns 200", func() {
				resp, err := statusClient.R().
					SetResult(&status.HealthResponse{}).
					Get("/health")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 200, resp.StatusCode())
				r := resp.Result().(*status.HealthResponse)
				assert.Equal(GinkgoT(), "UP", r.Status)
				assert.Equal(GinkgoT(), 2, len(r.Components)) // db, redis
				for _, hr := range r.Components {
					assert.Equal(GinkgoT(), "UP", hr.Status)
					assert.Nil(GinkgoT(), hr.Error)
				}
			})
		})

		Context("DOWN", func() {
			var a *app.Application
			var statusClient *resty.Client

			BeforeAll(func() {
				status.TestIndicators = []*health.Indicator{
					{
						Name: "always failed",
						Check: func() error {
							return errors.New("always failed")
						},
					},
				}
				helper.InitDB(true, nil)
				a = utils.Must(helper.Start(nil))
				statusClient = helper.StatusClient()
			})

			AfterAll(func() {
				status.TestIndicators = nil
				a.Stop()
			})

			It("should return 503", func() {
				resp, err := statusClient.R().
					SetError(&status.HealthResponse{}).
					Get("/health")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 503, resp.StatusCode())
				r := resp.Error().(*status.HealthResponse)
				assert.Equal(GinkgoT(), "DOWN", r.Status)
			})

		})
	})

})

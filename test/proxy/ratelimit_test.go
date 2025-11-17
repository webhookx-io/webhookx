package proxy

import (
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("rate-limit", Ordered, func() {

	Context("sanity", func() {

		var proxyClient *resty.Client
		var app *app.Application

		entitiesConfig := helper.EntitiesConfig{
			Sources: []*entities.Source{
				factory.SourceP(func(o *entities.Source) {
					o.RateLimit = &entities.RateLimit{
						Quota:  3,
						Period: 3,
					}
				}),
			},
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			resp, err := proxyClient.R().
				SetBody(`{ "event_type": "foo.bar", "data": { "key": "value" }}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.Equal(GinkgoT(), "3", resp.Header().Get("X-RateLimit-Limit"))
			assert.Equal(GinkgoT(), "2", resp.Header().Get("X-RateLimit-Remaining"))
			assert.True(GinkgoT(), utils.Must(strconv.Atoi(resp.Header().Get("X-RateLimit-Reset"))) > 0)

			resp, err = proxyClient.R().
				SetBody(`{ "event_type": "foo.bar", "data": { "key": "value" }}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.Equal(GinkgoT(), "3", resp.Header().Get("X-RateLimit-Limit"))
			assert.Equal(GinkgoT(), "1", resp.Header().Get("X-RateLimit-Remaining"))
			assert.True(GinkgoT(), resp.Header().Get("X-RateLimit-Reset") != "")

			resp, err = proxyClient.R().
				SetBody(`{ "event_type": "foo.bar", "data": { "key": "value" }}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.Equal(GinkgoT(), "3", resp.Header().Get("X-RateLimit-Limit"))
			assert.Equal(GinkgoT(), "0", resp.Header().Get("X-RateLimit-Remaining"))
			assert.True(GinkgoT(), resp.Header().Get("X-RateLimit-Reset") != "")

			resp, err = proxyClient.R().
				SetBody(`{ "event_type": "foo.bar", "data": { "key": "value" }}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 429, resp.StatusCode())
			assert.Equal(GinkgoT(), "{\"message\":\"rate limit exceeded\"}", string(resp.Body()))
			assert.Equal(GinkgoT(), "3", resp.Header().Get("X-RateLimit-Limit"))
			assert.Equal(GinkgoT(), "0", resp.Header().Get("X-RateLimit-Remaining"))
			assert.True(GinkgoT(), resp.Header().Get("X-RateLimit-Reset") != "")
			assert.True(GinkgoT(), utils.Must(strconv.Atoi(resp.Header().Get("Retry-After"))) > 0)

			time.Sleep(time.Second * 3)
			resp, err = proxyClient.R().
				SetBody(`{ "event_type": "foo.bar", "data": { "key": "value" }}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			assert.Equal(GinkgoT(), "3", resp.Header().Get("X-RateLimit-Limit"))
			assert.Equal(GinkgoT(), "2", resp.Header().Get("X-RateLimit-Remaining"))
			assert.True(GinkgoT(), utils.Must(strconv.Atoi(resp.Header().Get("X-RateLimit-Reset"))) > 0)
		})

	})

})

package clustering

import (
	"context"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("clustering", Ordered, func() {

	Context("clustering", func() {
		var adminClient *resty.Client
		var proxyClient *resty.Client

		var cp *app.Application
		var worker1 *app.Application
		var worker2 *app.Application
		var proxy *app.Application
		var db *db.DB

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources:   []*entities.Source{factory.Source()},
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			adminClient = helper.AdminClient()
			proxyClient = helper.ProxyClient()

			cp = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_STATUS_LISTEN": "off",
				"WEBHOOKX_ROLE":          "cp",
				"WEBHOOKX_LOG_FILE":      "webhookx-cp.log",
			}))
			worker1 = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_STATUS_LISTEN": "off",
				"WEBHOOKX_ROLE":          "dp_worker",
				"WEBHOOKX_LOG_FILE":      "webhookx-worker1.log",
			}))
			worker2 = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_STATUS_LISTEN": "off",
				"WEBHOOKX_ROLE":          "dp_worker",
				"WEBHOOKX_LOG_FILE":      "webhookx-worker2.log",
			}))
			proxy = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_STATUS_LISTEN": "off",
				"WEBHOOKX_ROLE":          "dp_proxy",
				"WEBHOOKX_LOG_FILE":      "webhookx-proxy.log",
			}))
		})

		BeforeEach(func() {
			err := db.Truncate("events")
			assert.Nil(GinkgoT(), err)
		})

		AfterAll(func() {
			cp.Stop()
			worker1.Stop()
			worker2.Stop()
			proxy.Stop()
		})

		It("events created from admin API must be delivered", func() {
			for i := 0; i < 100; i++ {
				resp, err := adminClient.R().
					SetBody(`{
				    "event_type": "foo.bar",
				    "data": {"key":"value"}
				}`).
					SetResult(entities.Event{}).
					Post("/workspaces/default/events")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 201, resp.StatusCode())
				result := resp.Result().(*entities.Event)
				assert.NotEmpty(GinkgoT(), result.ID)
				assert.Equal(GinkgoT(), "foo.bar", result.EventType)
				assert.Equal(GinkgoT(), `{"key":"value"}`, string(result.Data))
			}

			assert.Eventually(GinkgoT(), func() bool {
				q := query.AttemptQuery{
					Status: utils.Pointer(entities.AttemptStatusSuccess),
				}

				n, err := db.Attempts.Count(context.TODO(), q.WhereMap())
				assert.NoError(GinkgoT(), err)
				return n == 100
			}, time.Second*3, time.Second)
		})

		It("events ingested from proxy API must be delivered", func() {
			resp, err := proxyClient.R().
				SetBody(`{
					    "event_type": "foo.bar",
					    "data": {"key": "value"}
					}`).
				Post("/")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Attempts.List(context.TODO(), &query.AttemptQuery{})
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusSuccess
			}, time.Second*3, time.Second)
		})

	})
})

func TestClustering(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Clustering Suite")
}

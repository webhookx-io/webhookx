package delivery

import (
	"context"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"github.com/webhookx-io/webhookx/worker/deliverer"
	"net/netip"
	"time"
)

type ResolverFunc func(ctx context.Context, network, host string) ([]netip.Addr, error)

func (fn ResolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return fn(ctx, network, host)
}

var _ = Describe("network acl", Ordered, func() {
	Context("acl", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Events = []string{"test1"}
				}),
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Events = []string{"test2"}
					o.Request.URL = "http://www.example.com"
				}),
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Events = []string{"test3"}
					o.Request.URL = "http://suspicious.webhookx.io"
				}),
				factory.EndpointP(func(o *entities.Endpoint) {
					o.Events = []string{"unicode-test"}
					o.Request.URL = "http://тест.foo.com"
				}),
			},
			Sources: []*entities.Source{factory.SourceP()},
		}

		var resolver = deliverer.DefaultResolver

		BeforeAll(func() {
			deliverer.DefaultResolver = ResolverFunc(func(ctx context.Context, network, host string) ([]netip.Addr, error) {
				if host == "suspicious.webhookx.io" {
					return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
				}
				return resolver.LookupNetIP(ctx, network, host)
			})

			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_WORKER_DELIVERER_ACL_DENY": "@default,*.example.com,xn--e1aybc.foo.com",
			}))

		})

		AfterAll(func() {
			deliverer.DefaultResolver = resolver
			app.Stop()
		})

		It("request denied", func() {
			err := helper.WaitForServer("0.0.0.0:9600", time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "test1","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				q := query.AttemptQuery{}
				q.EventId = &eventId
				list, err := db.Attempts.List(context.TODO(), &q)
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusFailure
			}, time.Second*5, time.Second)

			// attempt.request
			assert.Equal(GinkgoT(), entities.AttemptErrorCodeDenied, *attempt.ErrorCode)
			assert.Equal(GinkgoT(), true, attempt.Exhausted)
			assert.Nil(GinkgoT(), attempt.Response)

			detail, err := db.AttemptDetails.Get(context.TODO(), attempt.ID)
			assert.NoError(GinkgoT(), err)
			assert.NotNil(GinkgoT(), detail.RequestHeaders)
			assert.NotNil(GinkgoT(), detail.RequestBody)
			assert.Nil(GinkgoT(), detail.ResponseHeaders)
			assert.Nil(GinkgoT(), detail.ResponseBody)
		})

		It("request denied by hostname", func() {
			err := helper.WaitForServer("0.0.0.0:9600", time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "test2","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				q := query.AttemptQuery{}
				q.EventId = &eventId
				list, err := db.Attempts.List(context.TODO(), &q)
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusFailure
			}, time.Second*5, time.Second)

			// attempt.request
			assert.Equal(GinkgoT(), entities.AttemptErrorCodeDenied, *attempt.ErrorCode)
			assert.Equal(GinkgoT(), true, attempt.Exhausted)
			assert.Nil(GinkgoT(), attempt.Response)
		})

		It("request denied by unicode hostname", func() {
			err := helper.WaitForServer("0.0.0.0:9600", time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "unicode-test","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				q := query.AttemptQuery{}
				q.EventId = &eventId
				list, err := db.Attempts.List(context.TODO(), &q)
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusFailure
			}, time.Second*5, time.Second)

			// attempt.request
			assert.Equal(GinkgoT(), entities.AttemptErrorCodeDenied, *attempt.ErrorCode)
			assert.Equal(GinkgoT(), true, attempt.Exhausted)
			assert.Nil(GinkgoT(), attempt.Response)
		})

		It("request denied by ip resolved by dns", func() {
			err := helper.WaitForServer("0.0.0.0:9600", time.Second)
			assert.NoError(GinkgoT(), err)

			resp, err := proxyClient.R().
				SetBody(`{"event_type": "test3","data": {"key": "value"}}`).
				Post("/")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
			eventId := resp.Header().Get(constants.HeaderEventId)

			var attempt *entities.Attempt
			assert.Eventually(GinkgoT(), func() bool {
				q := query.AttemptQuery{}
				q.EventId = &eventId
				list, err := db.Attempts.List(context.TODO(), &q)
				if err != nil || len(list) == 0 {
					return false
				}
				attempt = list[0]
				return attempt.Status == entities.AttemptStatusFailure
			}, time.Second*5, time.Second)

			// attempt.request
			assert.Equal(GinkgoT(), entities.AttemptErrorCodeDenied, *attempt.ErrorCode)
			assert.Equal(GinkgoT(), true, attempt.Exhausted)
			assert.Nil(GinkgoT(), attempt.Response)
		})
	})
})

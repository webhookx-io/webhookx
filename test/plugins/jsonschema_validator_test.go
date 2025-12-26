package plugins

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
)

var jsonString = `{
	"type": "object",
	"required": ["id", "amount", "currency"],
	"properties": {
		"id": {
			"type": "string"
		},
		"amount": {
			"type": "integer",
			"minimum": 1
		},
		"currency": {
			"type": "string",
			"maxLength": 6,
			"minLength": 3
		}
	}
}`

var _ = Describe("jsonschema-validator", Ordered, func() {

	Context("schema string", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint()},
			Sources:   []*entities.Source{factory.Source()},
		}
		entitiesConfig.Sources[0].Plugins = []*entities.Plugin{
			factory.Plugin("jsonschema-validator",
				factory.WithPluginConfig(jsonschema_validator.Config{
					Draft:         "6",
					DefaultSchema: jsonString,
					Schemas: map[string]*jsonschema_validator.Schema{
						"charge.succeeded": {
							Schema: jsonString,
						},
						"reuse.default_schema": nil,
					},
				}),
			),
		}

		BeforeAll(func() {
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))

			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("sanity", func() {
			body := `{"event_type": "charge.succeeded","data": {"id": "ch_1234567890","amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(200))
			// get event from db
			var event *entities.Event
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Events.List(context.TODO(), &query.EventQuery{})
				if err != nil || len(list) != 1 {
					return false
				}
				event = list[0]
				return true
			}, time.Second*5, time.Second)
			assert.Equal(GinkgoT(), "charge.succeeded", event.EventType)
			assert.JSONEq(GinkgoT(), `{"id": "ch_1234567890","amount": 1000,"currency": "usd"}`, string(event.Data))
		})

		It("sanity if undeclared event type", func() {
			body := `{"event_type": "unknown.event", "data":{"foo": "bar"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(200))

			// get event from db
			var event *entities.Event
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Events.List(context.TODO(), &query.EventQuery{})
				if err != nil || len(list) == 0 {
					return false
				}
				for _, item := range list {
					if item.EventType == "unknown.event" {
						event = item
						return true
					}
				}
				return false
			}, time.Second*5, time.Second)
			assert.Equal(GinkgoT(), "unknown.event", event.EventType)
			assert.JSONEq(GinkgoT(), `{"foo": "bar"}`, string(event.Data))
		})

		It("sanity if reuse default_schema", func() {
			body := `{"event_type": "reuse.default_schema","data": {"id": "ch_1234567890","amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(200))
			// get event from db
			var event *entities.Event
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Events.List(context.TODO(), &query.EventQuery{})
				if err != nil || len(list) == 0 {
					return false
				}
				for _, item := range list {
					if item.EventType == "reuse.default_schema" {
						event = item
						return true
					}
				}
				return true
			}, time.Second*5, time.Second)
			assert.Equal(GinkgoT(), "reuse.default_schema", event.EventType)
			assert.JSONEq(GinkgoT(), `{"id": "ch_1234567890","amount": 1000,"currency": "usd"}`, string(event.Data))
		})

		It("invalid event - missing required field", func() {
			body := `{"event_type": "charge.succeeded","data": {"amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"Request Validation","error":{"message":"request validation","fields":{"id":"required field missing"}}}`))
		})

		It("invalid event - field type mismatch", func() {
			body := `{"event_type": "charge.succeeded","data": {"id": "ch_1234567890","amount": "1000","currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")

			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"Request Validation","error":{"message":"request validation","fields":{"amount":"value must be an integer"}}}`))
		})

		It("invalid event - reuse default schema", func() {
			body := `{"event_type": "reuse.default_schema","data": {"id": "ch_1234567890","amount": "1000","currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")

			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"Request Validation","error":{"message":"request validation","fields":{"amount":"value must be an integer"}}}`))
		})
	})
})

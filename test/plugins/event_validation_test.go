package plugins

import (
	"context"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/plugins/event-validation"
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

var _ = Describe("event-validation", Ordered, func() {

	version := []string{"draft-04", "draft-06", "draft-07", "draft-2019-09", "draft-2020-12", "openapi-3.0"}
	for _, version := range version {
		Context("version: "+version, func() {
			var proxyClient *resty.Client

			var app *app.Application
			var db *db.DB

			entitiesConfig := helper.TestEntities{
				Endpoints: []*entities.Endpoint{factory.Endpoint()},
				Sources: []*entities.Source{
					factory.Source(),
					factory.Source(func(o *entities.Source) {
						o.Config.HTTP.Path = "/invalid-version"
					}),
					factory.Source(func(o *entities.Source) {
						o.Config.HTTP.Path = "/invalid-schema"
					}),
				},
			}
			entitiesConfig.Sources[0].Plugins = []*entities.Plugin{
				factory.Plugin("event-validation",
					factory.WithPluginConfig(event_validation.Config{
						VerboseResponse: true,
						Version:         version,
						Schemas: map[string]*string{
							"charge.succeeded": utils.Pointer(jsonString),
						},
					}),
				),
			}
			entitiesConfig.Sources[1].Plugins = []*entities.Plugin{
				factory.Plugin("event-validation",
					factory.WithPluginConfig(event_validation.Config{
						VerboseResponse: true,
						Version:         "unknown",
						DefaultSchema: utils.Pointer(jsonString),
					}),
				),
			}
			entitiesConfig.Sources[2].Plugins = []*entities.Plugin{
				factory.Plugin("event-validation",
					factory.WithPluginConfig(event_validation.Config{
						VerboseResponse: true,
						Version:         "openapi-3.0",
						DefaultSchema: utils.Pointer("{test}"),
					}),
				),
			}

			BeforeAll(func() {
				db = helper.InitDB(true, &entitiesConfig)
				proxyClient = helper.ProxyClient()
				app = helper.MustStart(map[string]string{})
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
				eventId := resp.Header().Get(constants.HeaderEventId)
				assert.NotEmpty(GinkgoT(), eventId)
				// get event from db
				event, err := db.Events.Get(context.TODO(), eventId)
				assert.Nil(GinkgoT(), err)
				assert.NotNil(GinkgoT(), event)
				assert.Equal(GinkgoT(), "charge.succeeded", event.EventType)
				assert.JSONEq(GinkgoT(), `{"id": "ch_1234567890","amount": 1000,"currency": "usd"}`, string(event.Data))
			})

			It("should succeed when event schema is not defined", func() {
				body := `{"event_type": "unknown.event", "data":{"foo": "bar"}}`
				resp, err := proxyClient.R().
					SetHeader("Content-Type", "application/json").
					SetBody(body).
					Post("/")
				Expect(err).To(BeNil())
				Expect(resp.StatusCode()).To(Equal(200))

				eventId := resp.Header().Get(constants.HeaderEventId)
				assert.NotEmpty(GinkgoT(), eventId)
				// get event from db
				event, err := db.Events.Get(context.TODO(), eventId)
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), "unknown.event", event.EventType)
				assert.JSONEq(GinkgoT(), `{"foo": "bar"}`, string(event.Data))
			})

			Context("errors", func() {
				if strings.Contains(version, "draft") {
					// jsonschema dialect
					It("should fail when missing required field", func() {
						body := `{"event_type": "charge.succeeded","data": {"amount": 1000,"currency": "usd"}}`
						resp, err := proxyClient.R().
							SetHeader("Content-Type", "application/json").
							SetBody(body).
							Post("/")
						Expect(err).To(BeNil())
						Expect(resp.StatusCode()).To(Equal(400))
						Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema","error":["at '': missing property 'id'"]}`))
					})

					It("should fail when field type mismatch", func() {
						body := `{"event_type": "charge.succeeded","data": {"id": "ch_1234567890","amount": "1000","currency": "usd"}}`
						resp, err := proxyClient.R().
							SetHeader("Content-Type", "application/json").
							SetBody(body).
							Post("/")
						Expect(err).To(BeNil())
						Expect(resp.StatusCode()).To(Equal(400))
						Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema","error":["at '/amount': got string, want integer"]}`))
					})

					It("should fail for multiple errors", func() {
						body := `{"event_type": "charge.succeeded","data": {"amount": "1000","currency": "usd"}}`
						resp, err := proxyClient.R().
							SetHeader("Content-Type", "application/json").
							SetBody(body).
							Post("/")
						Expect(err).To(BeNil())
						Expect(resp.StatusCode()).To(Equal(400))
						Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema","error":["at '': missing property 'id'","at '/amount': got string, want integer"]}`))
					})
				} else {
					// openapi dialect
					It("should fail when missing required field", func() {
						body := `{"event_type": "charge.succeeded","data": {"amount": 1000,"currency": "usd"}}`
						resp, err := proxyClient.R().
							SetHeader("Content-Type", "application/json").
							SetBody(body).
							Post("/")
						Expect(err).To(BeNil())
						Expect(resp.StatusCode()).To(Equal(400))
						Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema","error":["at 'id': property \"id\" is missing"]}`))
					})

					It("should fail when field type mismatch", func() {
						body := `{"event_type": "charge.succeeded","data": {"id": "ch_1234567890","amount": "1000","currency": "usd"}}`
						resp, err := proxyClient.R().
							SetHeader("Content-Type", "application/json").
							SetBody(body).
							Post("/")
						Expect(err).To(BeNil())
						Expect(resp.StatusCode()).To(Equal(400))
						Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema","error":["at 'amount': value must be an integer"]}`))
					})

					It("should fail for multiple errors", func() {
						body := `{"event_type": "charge.succeeded","data": {"amount": "1000","currency": "usd"}}`
						resp, err := proxyClient.R().
							SetHeader("Content-Type", "application/json").
							SetBody(body).
							Post("/")
						Expect(err).To(BeNil())
						Expect(resp.StatusCode()).To(Equal(400))
						Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema","error":["at 'amount': value must be an integer","at 'id': property \"id\" is missing"]}`))
					})
				}

				It("should return 500 when config.version is invalid", func() {
					resp, err := proxyClient.R().
						SetHeader("Content-Type", "application/json").
						SetBody("{}").
						Post("/invalid-version")
					Expect(err).To(BeNil())
					Expect(resp.StatusCode()).To(Equal(500))
					Expect(string(resp.Body())).To(Equal(`{"message":"internal error"}`))
				})

				It("should return 500 when plugin event's schema is invalid", func() {
					resp, err := proxyClient.R().
						SetHeader("Content-Type", "application/json").
						SetBody("{}").
						Post("/invalid-schema")
					Expect(err).To(BeNil())
					Expect(resp.StatusCode()).To(Equal(500))
					Expect(string(resp.Body())).To(Equal(`{"message":"internal error"}`))
				})
			})

		})

	}

	Context("verbose_response", func() {
		var proxyClient *resty.Client

		var app *app.Application

		BeforeAll(func() {
			_ = helper.InitDB(true, &helper.TestEntities{
				Sources: []*entities.Source{factory.Source(func(source *entities.Source) {
					source.Plugins = []*entities.Plugin{
						factory.Plugin("event-validation",
							factory.WithPluginConfig(event_validation.Config{
								VerboseResponse: false,
								Version:         "draft-04",
								Schemas: map[string]*string{
									"charge.succeeded": utils.Pointer(jsonString),
								},
							})),
					}
				})},
			})
			proxyClient = helper.ProxyClient()
			app = helper.MustStart(map[string]string{})
			err := helper.WaitForServer(helper.ProxyHttpURL, time.Second)
			assert.NoError(GinkgoT(), err)
		})

		AfterAll(func() {
			app.Stop()
		})

		It("should not return detailed error in response", func() {
			body := `{"event_type": "charge.succeeded","data": {"amount": 1000,"currency": "usd"}}`
			resp, err := proxyClient.R().
				SetHeader("Content-Type", "application/json").
				SetBody(body).
				Post("/")
			Expect(err).To(BeNil())
			Expect(resp.StatusCode()).To(Equal(400))
			Expect(string(resp.Body())).To(Equal(`{"message":"event data does not conform to schema"}`))
		})
	})
})

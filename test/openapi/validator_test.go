package openapi

import (
	"encoding/json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
)

func TestOpenAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenAPI Suite")
}

var _ = Describe("Schema", Ordered, func() {
	Context("sanity", func() {
		BeforeAll(func() {
			openapi.LoadOpenAPI(webhookx.OpenAPI)
		})

		It("new default entity", func() {
			var endpoint entities.Endpoint

			assert.NotPanics(GinkgoT(), func() {
				endpoint = openapi.NewEntity(func(t *entities.Endpoint) {
					t.ID = utils.KSUID()
				})
			})
			assert.Equal(GinkgoT(), true, endpoint.Enabled)
			assert.Equal(GinkgoT(), int64(10000), endpoint.Request.Timeout)
			assert.Equal(GinkgoT(), entities.RetryStrategyFixed, endpoint.Retry.Strategy)
			assert.Equal(GinkgoT(), []int64{0, 60, 3600}, endpoint.Retry.Config.Attempts)
		})

		It("validate entity", func() {
			endpoint := factory.Endpoint(
				factory.WithEndpointID(utils.KSUID()),
				factory.WithEndpointName("test"),
			)
			err := openapi.Validate(&endpoint)
			assert.Nil(GinkgoT(), err)
		})
	})

	Context("errors", func() {
		BeforeAll(func() {
			openapi.LoadOpenAPI(webhookx.OpenAPI)
		})
		It("panic if entity is not a schema", func() {
			type PanicExample struct {
				Name string `json:"name"`
			}

			assert.Panics(GinkgoT(), func() {
				openapi.NewEntity(func(a *PanicExample) {
					a.Name = "panic"
				})
			})
		})

		It("failed if entity is invalid", func() {
			endpoint := openapi.NewEntity(func(t *entities.Endpoint) {
				t.ID = utils.KSUID()
			})
			err := openapi.Validate(&endpoint)
			assert.NotNil(GinkgoT(), err)

			result, _ := json.Marshal(err)
			assert.Equal(GinkgoT(),
				`{"message":"request validation","fields":{"@body":{"request":{"method":["value is not one of the allowed values [\"GET\",\"POST\",\"PUT\",\"DELETE\",\"PATCH\"]"],"url":["property \"url\" is missing"]}}}}`,
				string(result),
			)
		})
	})
})

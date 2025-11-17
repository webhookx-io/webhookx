package delivery

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"testing"
)

var _ = Describe("schemas", Ordered, func() {

	Context("Endpoint", func() {
		var schema *openapi3.Schema
		BeforeAll(func() {
			entities.LoadOpenAPI(webhookx.OpenAPI)
			schema = entities.LookupSchema("Endpoint")
		})

		It("errors", func() {
			tests := []struct {
				name       string
				data       map[string]interface{}
				feildsJSON string
			}{
				{
					name: "request.url is emtpy",
					data: map[string]interface{}{
						"request": map[string]interface{}{
							"url": "",
						},
					},
					feildsJSON: `{"request":{"url":"minimum string length is 1"}}`,
				},
				{
					name: "request.timeout is negative",
					data: map[string]interface{}{
						"request": map[string]interface{}{
							"url":     "http://example.com",
							"timeout": -1,
						},
					},
					feildsJSON: `{"request":{"timeout":"number must be at least 0"}}`,
				},
				{
					name: "unknown retry.strategy",
					data: map[string]interface{}{
						"request": map[string]interface{}{
							"url": "http://example.com",
						},
						"retry": map[string]interface{}{
							"strategy": "unknown",
						},
					},
					feildsJSON: `{"retry":{"strategy":"value is not one of the allowed values [\"fixed\"]"}}`,
				},
				{
					name: "retry.config.attempts is empty list",
					data: map[string]interface{}{
						"request": map[string]interface{}{
							"url": "http://example.com",
						},
						"retry": map[string]interface{}{
							"strategy": "fixed",
							"config": map[string]interface{}{
								"attempts": []int{},
							},
						},
					},
					feildsJSON: `{"retry":{"config":{"attempts":"minimum number of items is 1"}}}`,
				},
				{
					name: "retry.config.attempts element is invalid",
					data: map[string]interface{}{
						"request": map[string]interface{}{
							"url": "http://example.com",
						},
						"retry": map[string]interface{}{
							"strategy": "fixed",
							"config": map[string]interface{}{
								"attempts": []interface{}{1, "str", true, 100},
							},
						},
					},
					feildsJSON: `{"retry":{"config":{"attempts":[null,"value must be an integer","value must be an integer"]}}}`,
				},
			}
			for _, test := range tests {
				err := openapi.Validate(schema, test.data)
				b, e := json.Marshal(err.(*errs.ValidateError).Fields)
				assert.NoError(GinkgoT(), e)
				assert.Equal(GinkgoT(), test.feildsJSON, string(b))
			}
		})
	})

	Context("Source", func() {
		var schema *openapi3.Schema
		BeforeAll(func() {
			entities.LoadOpenAPI(webhookx.OpenAPI)
			schema = entities.LookupSchema("Source")
		})

		It("errors", func() {
			tests := []struct {
				name       string
				data       map[string]interface{}
				feildsJSON string
			}{
				{
					name:       "missing requires fields",
					data:       map[string]interface{}{},
					feildsJSON: `{"config":"required field missing"}`,
				},
				{
					name: "config.http.methods is empty",
					data: map[string]interface{}{
						"type": "http",
						"config": map[string]interface{}{
							"http": map[string]interface{}{
								"path":    "/",
								"methods": []interface{}{},
							},
						},
					},
					feildsJSON: `{"config":{"http":{"methods":"minimum number of items is 1"}}}`,
				},
				{
					name: "config.http.methods element is invalid",
					data: map[string]interface{}{
						"type": "http",
						"config": map[string]interface{}{
							"http": map[string]interface{}{
								"path":    "/",
								"methods": []interface{}{"unknown"},
							},
						},
					},
					feildsJSON: `{"config":{"http":{"methods":["value is not one of the allowed values [\"GET\",\"POST\",\"PUT\",\"DELETE\",\"PATCH\"]"]}}}`,
				},
				{
					name: "config.http.response.code is invalid",
					data: map[string]interface{}{
						"type": "http",
						"config": map[string]interface{}{
							"http": map[string]interface{}{
								"path":    "/",
								"methods": []interface{}{"POST"},
								"response": map[string]interface{}{
									"code":         600,
									"content_type": "application/json",
								},
							},
						},
					},
					feildsJSON: `{"config":{"http":{"response":{"code":"number must be at most 599"}}}}`,
				},
			}
			for _, test := range tests {
				err := openapi.Validate(schema, test.data)
				b, e := json.Marshal(err.(*errs.ValidateError).Fields)
				assert.NoError(GinkgoT(), e)
				assert.Equal(GinkgoT(), test.feildsJSON, string(b))
			}
		})
	})

	Context("Configuration", func() {
		var schema *openapi3.Schema
		BeforeAll(func() {
			entities.LoadOpenAPI(webhookx.OpenAPI)
			schema = entities.LookupSchema("Configuration")
		})

		It("errors", func() {
			tests := []struct {
				name       string
				data       map[string]interface{}
				feildsJSON string
			}{
				{
					name: "invalid",
					data: map[string]interface{}{
						"endpoints": []interface{}{
							map[string]interface{}{},
							map[string]interface{}{
								"request": map[string]interface{}{"url": "http://example.com"},
							},
							map[string]interface{}{},
						},
						"sources": []interface{}{
							map[string]interface{}{},
							map[string]interface{}{
								"type": "http",
								"config": map[string]interface{}{
									"http": map[string]interface{}{
										"path":    "/",
										"methods": []interface{}{"POST"},
									},
								},
							},
							map[string]interface{}{},
						},
					},
					feildsJSON: `{"endpoints":[{"request":{"url":"required field missing"}},null,{"request":{"url":"required field missing"}}],"sources":[{"config":"required field missing"},null,{"config":"required field missing"}]}`,
				},
			}
			for _, test := range tests {
				err := openapi.Validate(schema, test.data)
				b, e := json.Marshal(err.(*errs.ValidateError).Fields)
				assert.NoError(GinkgoT(), e)
				assert.Equal(GinkgoT(), test.feildsJSON, string(b))
			}
		})
	})
})

func TestProxy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Suite")
}

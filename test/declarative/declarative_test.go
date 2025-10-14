package declarative

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"os"
	"testing"
)

var (
	malformedYAML = `
webhookx is coolest!
`
	invalidYAML = `
sources:
  - name: default-source
    path: /
    enabled: ok
`

	invalidEndpointYAML = `
endpoints:
  - name: default-endpoint
    events: [ "charge.succeeded" ]
`

	unknownPluginYAML = `
endpoints:
  - name: default-endpoint
    request:
      url: https://httpbin.org/anything
      method: POST
    events: [ "charge.succeeded" ]
    plugins:
      - name: foo
`

	invalidSourcePluginJSONSchemaJSONYAML = `
sources:
  - name: default-source
    path: /
    methods: ["POST"]
    plugins:
      - name: "jsonschema-validator"
        config:
          schemas:
            charge.succeed:
              json: '%s'
`

	invalidSourcePluginJSONSchemaFileYAML = `
sources:
  - name: default-source
    path: /
    methods: ["POST"]
    plugins:
      - name: "jsonschema-validator"
        config:
          schemas:
            charge.succeed:
              file: "%s"
`
	invalidSourcePluginJSONSchemaURLYAML = `
sources:
  - name: default-source
    path: /
    methods: ["POST"]
    plugins:
      - name: "jsonschema-validator"
        config:
          schemas:
            charge.succeed:
              url: "http://localhost/charge.succeed.json"
`
)

var _ = Describe("Declarative", Ordered, func() {
	var app *app.Application
	var adminClient *resty.Client

	BeforeAll(func() {
		helper.InitDB(true, nil)
		app = utils.Must(helper.Start(map[string]string{
			"WEBHOOKX_ADMIN_LISTEN": "0.0.0.0:8080",
		}))
		adminClient = helper.AdminClient()
	})

	AfterAll(func() {
		app.Stop()
	})

	Context("Admin", func() {
		It("sanity", func() {
			yaml, err := os.ReadFile("../fixtures/webhookx.yml")
			assert.Nil(GinkgoT(), err)

			resp, err := adminClient.R().
				SetBody(string(yaml)).
				Post("/workspaces/default/config/sync")
			fmt.Print(string(resp.Body()))
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
		})

		Context("errors", func() {
			It("should return 400 for malformed yaml", func() {
				resp, err := adminClient.R().
					SetBody(malformedYAML).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})
			It("should return 400 for invalid yaml", func() {
				resp, err := adminClient.R().
					SetBody(invalidYAML).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})
			It("should return 400 for invalid endpoint yaml", func() {
				resp, err := adminClient.R().
					SetBody(invalidEndpointYAML).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"endpoints":[{"request":{"url":"required field missing"}}]}}}`,
					string(resp.Body()))
			})
			It("should return 400 for unknown plugin", func() {
				resp, err := adminClient.R().
					SetBody(unknownPluginYAML).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"Request Validation","error":{"message":"request validation","fields":{"name":"unknown plugin name 'foo'"}}}`, string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator plugin config json string", func() {
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaJSONYAML, "invalid jsonstring")).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"schemas[charge.succeed]":{"json":"value must be a valid json string"}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator plugin config jsonschema", func() {
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaJSONYAML, `{"type":"invalidObject"}`)).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"schemas[charge.succeed]":{"json":"invalid jsonschema: unsupported 'type' value \"invalidObject\""}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator plugin config file", func() {
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaFileYAML, "./notexist.json")).
					Post("/workspaces/default/config/sync")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"schemas[charge.succeed]":{"file":"value must be a valid exist file"}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator config file content", func() {
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaFileYAML, "../fixtures/jsonschema/invalid.json")).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"schemas[charge.succeed]":{"file":"the content must be a valid json string"}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid source plugin config url", func() {
				resp, err := adminClient.R().
					SetBody(invalidSourcePluginJSONSchemaURLYAML).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"schemas[charge.succeed]":{"url":"failed to fetch schema: Get \"http://localhost/charge.succeed.json\": dial tcp [::1]:80: connect: connection refused"}}}}}`,
					string(resp.Body()))
			})
		})
	})
})

func TestDeclarative(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Declarative Suite")
}

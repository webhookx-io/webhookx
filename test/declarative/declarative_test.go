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

	invalidSourcePluginJSONSchemaConfigYAML = `
sources:
  - name: default-source
    path: /
    methods: ["POST"]
    plugins:
      - name: "jsonschema-validator"
        config:
          default_schema: |
            %s
          schemas:
            charge.succeed:
              schema: |
                %s
`

	invalidSourcePluginJSONSchemaJSONYAML = `
sources:
  - name: default-source
    path: /
    methods: ["POST"]
    plugins:
      - name: "jsonschema-validator"
        config:
          draft: "6"
          default_schema: |
            %s
          schemas:
            charge.succeed:
              schema: |
                %s
            reuse.default_schema:
`
)

var _ = Describe("Declarative", Ordered, func() {
	var app *app.Application
	var adminClient *resty.Client

	BeforeAll(func() {
		helper.InitDB(true, nil)
		app = utils.Must(helper.Start(map[string]string{}))
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

			It("should return 400 for invalid jsonschema-validator plugin config", func() {
				resp, err := adminClient.R().
					SetBody(
						fmt.Sprintf(invalidSourcePluginJSONSchemaConfigYAML, "invalid jsonschema", "invalid jsonschema"),
					).
					Post("/workspaces/default/config/sync")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"default_schema":"value must be a valid json string","draft":"required field missing","schemas[charge.succeed]":{"schema":"value must be a valid json string"}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator plugin config jsonschema string", func() {
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaJSONYAML,
						`{"type": "invlidObject","properties": {"id": { "type": "string"}}}`,
						`{"type": "object","properties": {"id": { "type": "number", "format":"invalid"}}}`)).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"default_schema":"unsupported 'type' value \"invlidObject\"","schemas[charge.succeed]":{"schema":"unsupported 'format' value \"invalid\""},"schemas[reuse.default_schema]":{"schema":"invalid due to reusing the default_schema definition"}}}}}`,
					string(resp.Body()))
			})
		})
	})
})

func TestDeclarative(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Declarative Suite")
}

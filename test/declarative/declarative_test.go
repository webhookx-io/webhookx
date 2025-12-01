package declarative

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator/jsonschema"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
	"os"
	"testing"
)

var (
	malformedYAML = `
webhookx is awesome 👻!
`
	invalidYAML = `
sources:
  - name: default-source
	config:
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
    config:
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
    config:
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
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"default_schema":"string doesn't match the format \"jsonschema\" (invalid character 'i' looking for beginning of value)","schemas":{"charge.succeed":{"schema":"string doesn't match the format \"jsonschema\" (invalid character 'i' looking for beginning of value)"}}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator plugin config", func() {
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaJSONYAML,
						`{"type": "invlidObject","properties": {"id": { "type": "string"}}}`,
						`{"type": "object","properties": {"id": { "type": "number", "format":"invalid"}}}`)).
					Post("/workspaces/default/config/sync")

				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"default_schema":"string doesn't match the format \"jsonschema\" (invalid character 'i' looking for beginning of value)","schemas":{"charge.succeed":{"schema":"string doesn't match the format \"jsonschema\" (invalid character 'i' looking for beginning of value)"}}}}}}`,
					string(resp.Body()))
			})

			It("should return 400 for invalid jsonschema-validator plugin config jsonschema invalid version", func() {
				schema := fmt.Sprintf(`"$schema": "https://json-schema.org/%s/schema"`, "invalid-version")
				resp, err := adminClient.R().
					SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaJSONYAML,
						fmt.Sprintf(`{%s,"type": "object"}`, schema),
						fmt.Sprintf(`{%s}`, schema))).
					Post("/workspaces/default/config/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(),
					`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"default_schema":"string doesn't match the format \"jsonschema\" (failing loading \"https://json-schema.org/invalid-version/schema\": invalid file url: https://json-schema.org/invalid-version/schema)","schemas":{"charge.succeed":{"schema":"string doesn't match the format \"jsonschema\" (failing loading \"https://json-schema.org/invalid-version/schema\": invalid file url: https://json-schema.org/invalid-version/schema)"},"reuse.default_schema":"Value is not nullable"}}}}}`,
					string(resp.Body()))
			})

			versions := []string{
				jsonschema.Draft_4,
				jsonschema.Draft_6,
				jsonschema.Draft_7,
				jsonschema.Draft_2019,
				jsonschema.Draft_2020,
			}
			for _, version := range versions {
				It(fmt.Sprintf("should return 400 for invalid jsonschema-validator plugin config jsonschema using drafts-%s", version), func() {
					schema := fmt.Sprintf(`"$schema": "https://json-schema.org/%s/schema"`, version)
					resp, err := adminClient.R().
						SetBody(fmt.Sprintf(invalidSourcePluginJSONSchemaJSONYAML,
							fmt.Sprintf(`{%s,"type": "invlidObject","properties": {"id": { "type": "string"}}}`, schema),
							fmt.Sprintf(`{%s,"type": "object","properties": {"id": { "type": "number", "format":"invalid"}}}`, schema))).
						Post("/workspaces/default/config/sync")
					assert.Nil(GinkgoT(), err)
					assert.Equal(GinkgoT(), 400, resp.StatusCode())
					assert.Equal(GinkgoT(),
						`{"message":"Request Validation","error":{"message":"request validation","fields":{"config":{"default_schema":"string doesn't match the format \"jsonschema\" ({\"type\":\"value is not one of the allowed values [\\\"array\\\",\\\"boolean\\\",\\\"integer\\\",\\\"null\\\",\\\"number\\\",\\\"object\\\",\\\"string\\\"] or value must be an array\"})","schemas":{"reuse.default_schema":"Value is not nullable"}}}}}`,
						string(resp.Body()))

				})
			}
		})
	})
})

func TestDeclarative(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Declarative Suite")
}

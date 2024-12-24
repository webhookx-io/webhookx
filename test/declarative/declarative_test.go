package declarative

import (
	"context"
	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"github.com/webhookx-io/webhookx/utils"
	"testing"
)

var (
	yaml = `
sources:
  - name: default-source
    path: /
    async: true
    methods: [ "POST" ]
    response:
      code: 200
      content_type: application/json
      body: '{"message": "OK"}'

endpoints:
  - name: default-endpoint
    request:
      timeout: 10000
      url: https://httpbin.org/anything
      method: POST
      headers:
        x-apikey: secret
    retry:
      strategy: fixed
      config:
        attempts: [0, 3600, 3600]
    events: [ "charge.succeeded" ]
    plugins:
      - name: webhookx-signature
        config:
          signing_secret: foo
`
	malformedYAML = `
webhookx is coolest!
`
	invalidYAML = `
sources:
  - name: default-source
    path: /
    enabled: ok
`

	unknownPluginYAML = `
endpoints:
  - name: default-endpoint
    events: [ "charge.succeeded" ]
    plugins:
      - name: foo
`
)

var _ = Describe("Declarative", Ordered, func() {
	var app *app.Application
	var adminClient *resty.Client
	var db *db.DB
	BeforeAll(func() {
		db = helper.InitDB(true, nil)
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
			resp, err := adminClient.R().
				SetBody(yaml).
				Post("/workspaces/default/sync")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())

			endpoint, err := db.Endpoints.Select(context.TODO(), "name", "default-endpoint")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "default-endpoint", *endpoint.Name)
			assert.Equal(GinkgoT(), true, endpoint.Enabled)
			assert.Equal(GinkgoT(), []string{"charge.succeeded"}, []string(endpoint.Events))
			assert.EqualValues(GinkgoT(), 10000, endpoint.Request.Timeout)
			assert.Equal(GinkgoT(), "https://httpbin.org/anything", endpoint.Request.URL)
			assert.Equal(GinkgoT(), "POST", endpoint.Request.Method)
			assert.Equal(GinkgoT(), "secret", endpoint.Request.Headers["x-apikey"])
			assert.EqualValues(GinkgoT(), "fixed", endpoint.Retry.Strategy)
			assert.EqualValues(GinkgoT(), []int64{0, 3600, 3600}, endpoint.Retry.Config.Attempts)

			source, err := db.Sources.Select(context.TODO(), "name", "default-source")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), "default-source", *source.Name)
			assert.Equal(GinkgoT(), true, source.Enabled)
			assert.Equal(GinkgoT(), "/", source.Path)
			assert.Equal(GinkgoT(), []string{"POST"}, []string(source.Methods))
			assert.Equal(GinkgoT(), 200, source.Response.Code)
			assert.Equal(GinkgoT(), "application/json", source.Response.ContentType)
			assert.Equal(GinkgoT(), `{"message": "OK"}`, source.Response.Body)

			plugins, err := db.Plugins.ListEndpointPlugin(context.TODO(), endpoint.ID)
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), 1, len(plugins))
			assert.Equal(GinkgoT(), "webhookx-signature", plugins[0].Name)
			assert.Equal(GinkgoT(), true, plugins[0].Enabled)
			assert.Equal(GinkgoT(), `{"signing_secret": "foo"}`, string(plugins[0].Config))
		})

		Context("errors", func() {
			It("should return 400 for malformed yaml", func() {
				resp, err := adminClient.R().
					SetBody(malformedYAML).
					Post("/workspaces/default/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})
			It("should return 400 for invalid yaml", func() {
				resp, err := adminClient.R().
					SetBody(invalidYAML).
					Post("/workspaces/default/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
			})
			It("should return 400 for unknown plugin", func() {
				resp, err := adminClient.R().
					SetBody(unknownPluginYAML).
					Post("/workspaces/default/sync")
				assert.Nil(GinkgoT(), err)
				assert.Equal(GinkgoT(), 400, resp.StatusCode())
				assert.Equal(GinkgoT(), `{"message":"invalid configuration: unknown plugin: foo"}`, string(resp.Body()))
			})
		})

		It("database entities should be deleted", func() {
			ws, err := db.Workspaces.GetDefault(context.TODO())
			assert.NoError(GinkgoT(), err)

			endpoint := factory.EndpointWS(ws.ID)
			err = db.Endpoints.Insert(context.TODO(), &endpoint)
			assert.NoError(GinkgoT(), err)

			source := factory.SourceP()
			source.WorkspaceId = ws.ID
			err = db.Sources.Insert(context.TODO(), source)
			assert.NoError(GinkgoT(), err)

			resp, err := adminClient.R().
				SetBody(yaml).
				Post("/workspaces/default/sync")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())

			e1, err := db.Endpoints.Get(context.TODO(), endpoint.ID)
			assert.NoError(GinkgoT(), err)
			assert.Nil(GinkgoT(), e1)

			e2, err := db.Sources.Get(context.TODO(), source.ID)
			assert.NoError(GinkgoT(), err)
			assert.Nil(GinkgoT(), e2)
		})

		It("id should not be changed", func() {
			resp, err := adminClient.R().
				SetBody(yaml).
				Post("/workspaces/default/sync")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())

			endpoint1, err := db.Endpoints.Select(context.TODO(), "name", "default-endpoint")
			assert.NoError(GinkgoT(), err)

			resp, err = adminClient.R().
				SetBody(yaml).
				Post("/workspaces/default/sync")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())

			endpoint2, err := db.Endpoints.Select(context.TODO(), "name", "default-endpoint")
			assert.NoError(GinkgoT(), err)

			assert.Equal(GinkgoT(), endpoint1.ID, endpoint2.ID)
		})
	})
})

func TestDeclarative(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Declarative Suite")
}

package declarative

import (
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
				Post("/workspaces/default/sync")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), 200, resp.StatusCode())
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
	})
})

func TestDeclarative(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Declarative Suite")
}

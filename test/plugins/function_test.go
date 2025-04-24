package plugins

import (
	"fmt"
	"github.com/webhookx-io/webhookx/plugins/function"
	"github.com/webhookx-io/webhookx/plugins/function/api"
	"github.com/webhookx-io/webhookx/test/helper/factory"
	"time"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var function_verify_signature = `
function handle() {
	var bytes = webhookx.utils.hmac('SHA-256', "my_secret", webhookx.request.getBody())
	var signature = webhookx.utils.encode('hex', bytes)
    var signatureHeader = webhookx.request.getHeader("X-Signature")
	if (!webhookx.utils.digestEqual(signature, signatureHeader)) {
		webhookx.response.exit(400, { 'Content-Type': 'application/json' }, { message: 'invalid signature' })
	}
	webhookx.log.debug('valid signature')
}
`

var _ = Describe("function", Ordered, func() {

	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		//var db *db.DB

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}
		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(
				factory.WithPluginSourceID(entitiesConfig.Sources[0].ID),
				factory.WithPluginName("function"),
				factory.WithPluginConfig(function.Config{
					Function: function_verify_signature,
				}),
			),
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{
				"WEBHOOKX_ADMIN_LISTEN":   "0.0.0.0:8080",
				"WEBHOOKX_PROXY_LISTEN":   "0.0.0.0:8081",
				"WEBHOOKX_WORKER_ENABLED": "true",
			}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("OK", func() {
			utilsAPI := api.NewUtilsAPI()
			body := `{"event_type": "foo.bar","data": {"key": "value"}}`
			signature := utilsAPI.Encode("hex", utilsAPI.Hmac("SHA-256", "my_secret", body))
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetHeader("X-Signature", signature).
					SetBody(body).
					Post("/")
				fmt.Println(string(resp.Body()))
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			matched, err := helper.FileHasLine("webhookx.log", "^.*valid signature$")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)
		})

		It("should return desired response for invalid signature", func() {
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetHeader("X-Signature", "test").
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/")
				return err == nil && resp.StatusCode() == 400 &&
					string(resp.Body()) == `{"message":"invalid signature"}` &&
					resp.Header().Get("Content-Type") == "application/json"
			}, time.Second*5, time.Second)
		})

	})
})

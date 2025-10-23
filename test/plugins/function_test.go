package plugins

import (
	"context"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/plugins/function"
	"github.com/webhookx-io/webhookx/plugins/function/sdk"
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
	if (!webhookx.utils.timingSafeEqual(signature, signatureHeader)) {
		webhookx.response.exit(400, { 'Content-Type': 'application/json' }, { message: 'invalid signature' })
	}
	webhookx.log.debug('valid signature')
	var obj = JSON.parse(webhookx.request.getBody())
    obj.data.foo = 'bar'
    webhookx.request.setBody(JSON.stringify(obj))
}
`

var _ = Describe("function", Ordered, func() {

	Context("sanity", func() {
		var proxyClient *resty.Client

		var app *app.Application
		var db *db.DB

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
			db = helper.InitDB(true, &entitiesConfig)
			proxyClient = helper.ProxyClient()

			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("OK", func() {
			utilsAPI := sdk.NewUtilsSDK()
			body := `{"event_type": "foo.bar","data": {"key": "value"}}`
			signature := utilsAPI.Encode("hex", utilsAPI.Hmac("SHA-256", "my_secret", body))
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetHeader("X-Signature", signature).
					SetBody(body).
					Post("/")
				return err == nil && resp.StatusCode() == 200
			}, time.Second*5, time.Second)

			matched, err := helper.FileHasLine("webhookx.log", "^.*valid signature$")
			assert.Nil(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, matched)

			// payload should be changed
			var event *entities.Event
			assert.Eventually(GinkgoT(), func() bool {
				list, err := db.Events.List(context.TODO(), &query.EventQuery{})
				if err != nil || len(list) != 1 {
					return false
				}
				event = list[0]
				return true
			}, time.Second*5, time.Second)
			assert.JSONEq(GinkgoT(), `{"foo": "bar", "key": "value"}`, string(event.Data))
		})

		It("should return desired response for invalid signature", func() {
			assert.Eventually(GinkgoT(), func() bool {
				resp, err := proxyClient.R().
					SetHeader("X-Signature", "fake").
					SetBody(`{"event_type": "foo.bar","data": {"key": "value"}}`).
					Post("/")
				return err == nil && resp.StatusCode() == 400 &&
					string(resp.Body()) == `{"message":"invalid signature"}` &&
					resp.Header().Get("Content-Type") == "application/json"
			}, time.Second*5, time.Second)
		})
	})
})

package plugins

import (
	"context"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/hello"
	"github.com/webhookx-io/webhookx/test/fixtures/plugins/outbound"
	"github.com/webhookx-io/webhookx/test/helper/factory"

	. "github.com/onsi/ginkgo/v2"
	"github.com/webhookx-io/webhookx/app"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/test/helper"
	"github.com/webhookx-io/webhookx/utils"
)

var _ = Describe("ordering", Ordered, func() {

	plugin.RegisterPlugin(plugin.TypeOutbound, "outbound", func() plugin.Plugin {
		return &outbound.OutboundPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeOutbound, "hello", func() plugin.Plugin {
		return &hello.HelloPlugin{}
	})
	Context("", func() {

		var app *app.Application

		entitiesConfig := helper.TestEntities{
			Endpoints: []*entities.Endpoint{factory.Endpoint(factory.WithEndpointPlugins(
				factory.Plugin("wasm"),
				factory.Plugin("webhookx-signature"),
				factory.Plugin("hello"),
				factory.Plugin("outbound"),
				factory.Plugin("a-disabled-plugin", func(o *entities.Plugin) { o.Enabled = false }),
			))},
			Sources: []*entities.Source{factory.Source(factory.WithSourcePlugins(
				factory.Plugin("basic-auth"),
				factory.Plugin("function"),
				factory.Plugin("hmac-auth"),
				factory.Plugin("connect-auth"),
				factory.Plugin("event-validation"),
				factory.Plugin("key-auth"),
			))},
		}

		BeforeAll(func() {
			helper.InitDB(true, &entitiesConfig)
			app = utils.Must(helper.Start(map[string]string{}))
		})

		AfterAll(func() {
			app.Stop()
		})

		It("inbound plugins should be executed in ordered", func() {
			iterator := plugins.LoadIterator()

			list := iterator.Iterate(context.TODO(), plugins.PhaseInbound, entitiesConfig.Sources[0].ID)
			names := make([]string, 0)
			for plugin := range list {
				names = append(names, plugin.Name())
			}
			expectedOrdered := []string{
				"basic-auth",
				"key-auth",
				"hmac-auth",
				"connect-auth",
				"event-validation",
				"function",
			}
			assert.Equal(GinkgoT(), expectedOrdered, names)
		})

		It("outbound plugins should be executed in ordered", func() {
			iterator := plugins.LoadIterator()

			list := iterator.Iterate(context.TODO(), plugins.PhaseOutbound, entitiesConfig.Endpoints[0].ID)
			names := make([]string, 0)
			for plugin := range list {
				names = append(names, plugin.Name())
			}
			expectedOrdered := []string{
				"outbound",
				"hello",
				"wasm",
				"webhookx-signature",
			}
			assert.Equal(GinkgoT(), expectedOrdered, names)
		})
	})
})

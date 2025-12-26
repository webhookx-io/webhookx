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

		entitiesConfig := helper.EntitiesConfig{
			Endpoints: []*entities.Endpoint{factory.EndpointP()},
			Sources:   []*entities.Source{factory.SourceP()},
		}

		entitiesConfig.Plugins = []*entities.Plugin{
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "basic-auth"
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "function"
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "hmac-auth"
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "connect-auth"
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "jsonschema-validator"
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "key-auth"
				o.SourceId = utils.Pointer(entitiesConfig.Sources[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "wasm"
				o.EndpointId = utils.Pointer(entitiesConfig.Endpoints[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "webhookx-signature"
				o.EndpointId = utils.Pointer(entitiesConfig.Endpoints[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "hello"
				o.EndpointId = utils.Pointer(entitiesConfig.Endpoints[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "outbound"
				o.EndpointId = utils.Pointer(entitiesConfig.Endpoints[0].ID)
			}),
			factory.PluginP(func(o *entities.Plugin) {
				o.Name = "a-disabled-plugin"
				o.Enabled = false
			}),
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
				"jsonschema-validator",
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

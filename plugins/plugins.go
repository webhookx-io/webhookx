package plugins

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	basic_auth "github.com/webhookx-io/webhookx/plugins/basic-auth"
	integration_auth "github.com/webhookx-io/webhookx/plugins/connect-auth"
	"github.com/webhookx-io/webhookx/plugins/event-validation"
	"github.com/webhookx-io/webhookx/plugins/function"
	hmac_auth "github.com/webhookx-io/webhookx/plugins/hmac-auth"
	key_auth "github.com/webhookx-io/webhookx/plugins/key-auth"
	"github.com/webhookx-io/webhookx/plugins/wasm"
	"github.com/webhookx-io/webhookx/plugins/webhookx_signature"
)

func LoadPlugins() {
	plugin.RegisterPlugin(plugin.TypeInbound, "function", func() plugin.Plugin {
		return &function.FunctionPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeOutbound, "wasm", func() plugin.Plugin {
		return &wasm.WasmPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeOutbound, "webhookx-signature", func() plugin.Plugin {
		return &webhookx_signature.SignaturePlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeInbound, "event-validation", func() plugin.Plugin {
		return &event_validation.EventValidationPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeInbound, "basic-auth", func() plugin.Plugin {
		return &basic_auth.BasicAuthPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeInbound, "key-auth", func() plugin.Plugin {
		return &key_auth.KeyAuthPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeInbound, "hmac-auth", func() plugin.Plugin {
		return &hmac_auth.HmacAuthPlugin{}
	})
	plugin.RegisterPlugin(plugin.TypeInbound, "connect-auth", func() plugin.Plugin {
		return &integration_auth.ConnectAuthPlugin{}
	})
}

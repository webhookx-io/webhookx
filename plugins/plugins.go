package plugins

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/function"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator"
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
	plugin.RegisterPlugin(plugin.TypeInbound, "jsonschema-validator", func() plugin.Plugin {
		return &jsonschema_validator.SchemaValidatorPlugin{}
	})
}

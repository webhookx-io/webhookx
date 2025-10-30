package plugins

import (
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/plugins/function"
	"github.com/webhookx-io/webhookx/plugins/jsonschema_validator"
	"github.com/webhookx-io/webhookx/plugins/wasm"
	"github.com/webhookx-io/webhookx/plugins/webhookx_signature"
)

func LoadPlugins() {
	plugin.RegisterPlugin(plugin.TypeInbound, "function", function.New)
	plugin.RegisterPlugin(plugin.TypeInbound, "jsonschema-validator", jsonschema_validator.New)
	plugin.RegisterPlugin(plugin.TypeOutbound, "wasm", wasm.New)
	plugin.RegisterPlugin(plugin.TypeOutbound, "webhookx-signature", webhookx_signature.New)
}

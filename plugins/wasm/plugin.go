package wasm

import (
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Config struct {
	File string            `json:"file"`
	Envs map[string]string `json:"envs"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("WasmPluginConfiguration")
}

type WasmPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *WasmPlugin) Name() string {
	return "wasm"
}

func (p *WasmPlugin) Priority() int {
	return -90
}

func (p *WasmPlugin) ExecuteOutbound(c *plugin.Context) error {
	ctx := c.Context()
	source, err := os.ReadFile(p.Config.File)
	if err != nil {
		return err
	}

	runtime := wazero.NewRuntime(ctx)
	defer func() { _ = runtime.Close(ctx) }()

	_, err = runtime.NewHostModuleBuilder("env").
		NewFunctionBuilder().WithFunc(Log).Export("log").
		NewFunctionBuilder().WithFunc(GetRequestJSON).Export("get_request_json").
		NewFunctionBuilder().WithFunc(SetRequestJSON).Export("set_request_json").
		Instantiate(ctx)
	if err != nil {
		return err
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	cfg := wazero.NewModuleConfig().WithStartFunctions("_initialize", "_start")
	for k, v := range p.Config.Envs {
		cfg = cfg.WithEnv(k, v)
	}
	mod, err := runtime.InstantiateWithConfig(ctx, source, cfg)
	if err != nil {
		return err
	}

	transform := mod.ExportedFunction("transform")
	if transform == nil {
		return fmt.Errorf("exported function 'transform' is not defined in module")
	}

	ctx = withContext(ctx, c)
	results, err := transform.Call(ctx)
	if err != nil {
		return err
	}
	if len(results) != 1 {
		return fmt.Errorf("exported function 'transform' must return exactly one result")
	}
	if results[0] != 1 {
		return fmt.Errorf("transform failed with value %d", results[0])
	}

	return nil
}

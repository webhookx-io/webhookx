package wasm

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
	"github.com/webhookx-io/webhookx/utils"
	"os"
)

type Config struct {
	File string            `json:"file" validate:"required"`
	Envs map[string]string `json:"envs"`
}

func (cfg *Config) Validate() error {
	return utils.Validate(cfg)
}

func (cfg *Config) ProcessDefault() {}

type WasmPlugin struct {
	types.BasePlugin

	cfg Config
}

func New() types.Plugin {
	plugin := &WasmPlugin{}
	plugin.Name = "wasm"
	return plugin
}

func (p *WasmPlugin) Execute(req *types.Request, _ *types.Context) error {
	source, err := os.ReadFile(p.cfg.File)
	if err != nil {
		return err
	}

	ctx := context.Background()
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
	for k, v := range p.cfg.Envs {
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

	ctx = withContext(ctx, req)
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

func (p *WasmPlugin) Config() types.PluginConfig {
	return &p.cfg
}

package wasm

import (
	"context"
	"fmt"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
	"os"
)

type Config struct {
	File string            `json:"file" validate:"required"`
	Envs map[string]string `json:"envs"`
}

type WasmPlugin struct {
	plugin.BasePlugin[Config]
}

func New(config []byte) (plugin.Plugin, error) {
	p := &WasmPlugin{}
	p.Name = "wasm"

	if config != nil {
		if err := p.UnmarshalConfig(config); err != nil {
			return nil, err
		}
	}

	return p, nil
}
func (p *WasmPlugin) ValidateConfig() error {
	return utils.Validate(p.Config)
}

func (p *WasmPlugin) ExecuteOutbound(req *plugin.OutboundRequest, _ *plugin.Context) error {
	source, err := os.ReadFile(p.Config.File)
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

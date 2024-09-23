package plugin

import (
	"encoding/json"
	"errors"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
	"github.com/webhookx-io/webhookx/pkg/plugin/webhookx_signature"
)

func New(name string) types.Plugin {
	switch name {
	case "webhookx-signature":
		return webhookx_signature.New()
	}
	return nil
}

func ExecutePlugin(plugin *entities.Plugin, req *types.Request, ctx *types.Context) error {
	instance := New(plugin.Name)
	if instance == nil {
		return errors.New("unknown plugin: " + plugin.Name)
	}
	err := json.Unmarshal(plugin.Config, instance.Config())
	if err != nil {
		return err
	}
	instance.Execute(req, ctx)
	return nil
}

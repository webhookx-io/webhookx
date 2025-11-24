package modules

import "github.com/webhookx-io/webhookx/config/types"

var _ types.Config = BaseConfig{}

type BaseConfig struct{}

func (c BaseConfig) PostProcess() error { return nil }
func (c BaseConfig) Validate() error    { return nil }

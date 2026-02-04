package plugin

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/mitchellh/mapstructure"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/utils"
)

// Configuration plugin configuration
type Configuration interface {
	Schema() *openapi3.Schema
}

type BasePlugin[T Configuration] struct {
	Config T
}

func (p *BasePlugin[T]) Init(config map[string]interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &p.Config,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(config)
}

func (p *BasePlugin[T]) Priority() int {
	return 0
}

func (p *BasePlugin[T]) GetConfig() map[string]interface{} {
	m, err := utils.StructToMap(p.Config)
	if err != nil {
		panic(err)
	}
	return m
}

func (p *BasePlugin[T]) ValidateConfig(config map[string]interface{}) error {
	err := openapi.Validate(p.Config.Schema(), config)
	if err != nil {
		return err
	}
	return nil
}

func (p *BasePlugin[T]) ExecuteInbound(c *Context) error {
	panic("not implemented")
}

func (p *BasePlugin[T]) ExecuteOutbound(c *Context) error {
	panic("not implemented")
}


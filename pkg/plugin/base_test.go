package plugin

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

type config struct {
}

func (c config) Schema() *openapi3.Schema {
	return &openapi3.Schema{}
}

type MyPlugin struct {
	BasePlugin[config]
}

func (m MyPlugin) Name() string {
	panic("my-plugin")
}

func Test(t *testing.T) {
	myPlugin := &MyPlugin{}
	assert.PanicsWithValue(t, "not implemented", func() { myPlugin.ExecuteInbound(context.TODO(), nil) })
	assert.PanicsWithValue(t, "not implemented", func() { myPlugin.ExecuteOutbound(context.TODO(), nil) })
}

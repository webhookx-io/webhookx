package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/secret"
)

type Config struct{}

func (c Config) Schema() *openapi3.Schema { return nil }

type MyPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *MyPlugin) Name() string { return "my-plugin" }

type MockProvider struct{}

func (p *MockProvider) GetValue(ctx context.Context, key string, properties map[string]string) (string, error) {
	if properties["error"] != "" {
		return "", errors.New(properties["error"])
	}
	return key, nil
}

func init() {
	plugin.RegisterPlugin(plugin.TypeInbound, "my-plugin", func() plugin.Plugin { return &MyPlugin{} })
}

func Test(t *testing.T) {
	plugin := &entities.Plugin{
		Name:    "my-plugin",
		Enabled: true,
		Config: map[string]interface{}{
			"key":           "value",
			"key-reference": "{secret://mock/value}",
			"list":          []interface{}{"value", "{secret://mock/value}"},
		},
	}
	sm := secret.NewManager(secret.Options{})
	sm.AddProvider("mock", &MockProvider{})

	iterator := NewIterator("")
	iterator.WithSecretManager(sm)
	err := iterator.LoadPlugins([]*entities.Plugin{plugin})
	assert.NoError(t, err)
	fmt.Println(err)
	b, err := json.Marshal(plugin.Config)
	assert.JSONEq(t,
		`{"key":"value","key-reference":"value","list":["value","value"]}`,
		string(b))
}

func TestError(t *testing.T) {
	sm := secret.NewManager(secret.Options{})
	sm.AddProvider("mock", &MockProvider{})

	tests := []struct {
		scenario  string
		plugin    *entities.Plugin
		expectErr error
	}{
		{
			scenario: "",
			plugin: &entities.Plugin{
				ID:      "id1",
				Name:    "my-plugin",
				Enabled: true,
				Config: map[string]interface{}{
					"key": "{secret://mock/value?error=test}",
				},
			},
			expectErr: fmt.Errorf(`plugin{id=id1} configuration reference resolve failed: property "key" resolve error: failed to resolve reference value '{secret://mock/value?error=test}': test`),
		},
		{
			scenario: "",
			plugin: &entities.Plugin{
				ID:      "id1",
				Name:    "my-plugin",
				Enabled: true,
				Config: map[string]interface{}{
					"list": []interface{}{
						"{secret://mock/value?error=test}",
					},
				},
			},
			expectErr: fmt.Errorf(`plugin{id=id1} configuration reference resolve failed: property "list.[0]" resolve error: failed to resolve reference value '{secret://mock/value?error=test}': test`),
		},
	}

	for _, test := range tests {
		iterator := NewIterator("")
		iterator.WithSecretManager(sm)
		err := iterator.LoadPlugins([]*entities.Plugin{test.plugin})
		assert.EqualError(t, err, test.expectErr.Error())
	}

}

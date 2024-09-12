package webhookx_signature

import (
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/pkg/plugin/types"
	"testing"
	"time"
)

func TestExecute(t *testing.T) {
	plugin := New()
	plugin.(*SignaturePlugin).ts = time.Unix(1726285679, 0)
	plugin.(*SignaturePlugin).cfg.Key = "QGvaZ0uPwA9nYi7jr31JtZn1EKK4pJpK"

	pluginReq := &types.Request{
		URL:     "https://example.com",
		Method:  "POST",
		Headers: make(map[string]string),
		Payload: []byte("foo"),
	}
	plugin.Execute(pluginReq, nil)

	assert.Equal(t, "https://example.com", pluginReq.URL)
	assert.Equal(t, "POST", pluginReq.Method)
	assert.Equal(t, []byte("foo"), pluginReq.Payload)
	assert.Equal(t, "v1=e2af2618d5ffd700eb369904b7237ec4ac7d37873cfe6654265af2e53b44da6b", pluginReq.Headers["webhookx-signature"])
	assert.Equal(t, "1726285679", pluginReq.Headers["webhookx-timestamp"])
}

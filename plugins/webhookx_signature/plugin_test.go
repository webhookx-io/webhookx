package webhookx_signature

import (
	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"testing"
	"time"
)

func TestExecute(t *testing.T) {
	p, err := New(nil)
	assert.Nil(t, err)
	p.(*SignaturePlugin).ts = time.Unix(1726285679, 0)
	p.(*SignaturePlugin).Config.SigningSecret = "QGvaZ0uPwA9nYi7jr31JtZn1EKK4pJpK"

	pluginReq := &plugin.Outbound{
		URL:     "https://example.com",
		Method:  "POST",
		Headers: make(map[string]string),
		Payload: "foo",
	}
	p.ExecuteOutbound(pluginReq, nil)

	assert.Equal(t, "https://example.com", pluginReq.URL)
	assert.Equal(t, "POST", pluginReq.Method)
	assert.Equal(t, "foo", pluginReq.Payload)
	assert.Equal(t, "v1=e2af2618d5ffd700eb369904b7237ec4ac7d37873cfe6654265af2e53b44da6b", pluginReq.Headers["webhookx-signature"])
	assert.Equal(t, "1726285679", pluginReq.Headers["webhookx-timestamp"])
}

package webhookx_signature

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

func TestExecute(t *testing.T) {
	p := new(SignaturePlugin)
	p.ts = time.Unix(1726285679, 0)
	p.Config.SigningSecret = "QGvaZ0uPwA9nYi7jr31JtZn1EKK4pJpK"

	r, err := http.NewRequest("POST", "https://example.com", nil)
	assert.NoError(t, err)

	c := plugin.NewContext(context.TODO(), r, nil)
	c.SetRequestBody([]byte("foo"))
	err = p.ExecuteOutbound(c)
	assert.NoError(t, err)



	assert.Equal(t, "https://example.com", c.Request.URL.String())
	assert.Equal(t, "POST", c.Request.Method)
	assert.Equal(t, "foo", string(c.GetRequestBody()))
	assert.Equal(t, "v1=e2af2618d5ffd700eb369904b7237ec4ac7d37873cfe6654265af2e53b44da6b", c.Request.Header.Get("webhookx-signature"))
	assert.Equal(t, "1726285679", c.Request.Header.Get("webhookx-timestamp"))
}

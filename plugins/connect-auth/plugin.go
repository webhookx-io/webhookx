package connect_auth

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins/connect-auth/verifier"
)

type Config struct {
	Provider       string                 `json:"provider"`
	ProviderConfig map[string]interface{} `json:"provider_config"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("ConnectAuthPluginConfiguration")
}

type ConnectAuthPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *ConnectAuthPlugin) Name() string {
	return "connect-auth"
}

func (p *ConnectAuthPlugin) Priority() int {
	return 106
}

func (p *ConnectAuthPlugin) ExecuteInbound(c *plugin.Context) error {
	v, ok := verifier.LoadVerifier(p.Config.Provider)
	if !ok {
		return errors.New("unknown provider: " + p.Config.Provider)
	}

	r := c.Request.Clone(context.TODO())
	r.Body = io.NopCloser(bytes.NewReader(c.GetRequestBody()))
	request := verifier.Request{
		R: r,
	}
	res, err := v.Verify(c.Context(), &request, p.Config.ProviderConfig)
	if err != nil {
		return err
	}

	if !res.Verified {
		c.JSON(401, types.ErrorResponse{Message: "Unauthorized"})
		return nil
	}

	if res.Response != nil {
		c.Response(res.Response.Headers, res.Response.StatusCode, res.Response.Body)
	}

	return nil
}

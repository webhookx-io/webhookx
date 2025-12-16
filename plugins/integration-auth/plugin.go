package integration_auth

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/http/response"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/plugins/integration-auth/verifier"
)

type Config struct {
	Provider       string                 `json:"provider"`
	ProviderConfig map[string]interface{} `json:"provider_config"`
}

func (c Config) Schema() *openapi3.Schema {
	return entities.LookupSchema("IntegrationAuthPluginConfiguration")
}

type IntegrationAuthPlugin struct {
	plugin.BasePlugin[Config]
}

func (p *IntegrationAuthPlugin) Name() string {
	return "integration-auth"
}

func (p *IntegrationAuthPlugin) ExecuteInbound(ctx context.Context, inbound *plugin.Inbound) (result plugin.InboundResult, err error) {
	v, ok := verifier.LoadVerifier(p.Config.Provider)
	if !ok {
		return result, errors.New("unknown provider: " + p.Config.Provider)
	}

	r := inbound.Request.Clone(ctx)
	r.Body = io.NopCloser(bytes.NewReader(inbound.RawBody))
	request := verifier.Request{
		R: r,
	}
	res, err := v.Verify(ctx, &request, p.Config.ProviderConfig)
	if err != nil {
		return result, err
	}

	if !res.Verified {
		response.JSON(inbound.Response, 401, types.ErrorResponse{Message: "Unauthorized"})
		result.Terminated = true
		return
	}

	if res.Response != nil {
		for k, v := range res.Response.Headers {
			inbound.Response.Header().Set(k, v)
		}
		inbound.Response.WriteHeader(res.Response.StatusCode)
		_, _ = inbound.Response.Write(res.Response.Body)
		result.Terminated = true
		return
	}

	result.Payload = inbound.RawBody
	return
}

package plugin

import (
	"context"
	"net/http"
)

type Plugin interface {
	// Name returns plugin's name
	Name() string

	// Init inits plugin with configuration
	Init(config map[string]interface{}) error

	// GetConfig returns plugin's configuration
	GetConfig() map[string]interface{}

	// ValidateConfig validates plugin's configuration
	ValidateConfig(config map[string]interface{}) error

	// ExecuteInbound executes inbound
	ExecuteInbound(ctx context.Context, inbound *Inbound) (InboundResult, error)

	// ExecuteOutbound executes outbound
	ExecuteOutbound(ctx context.Context, outbound *Outbound) error
}

func New(name string) (Plugin, bool) {
	r := GetRegistration(name)
	if r == nil {
		return nil, false
	}
	return r.Factory(), true
}

type Outbound struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Payload string            `json:"payload"`
}

type Inbound struct {
	Request  *http.Request
	Response http.ResponseWriter
	RawBody  []byte
}

type InboundResult struct {
	Terminated bool
	Payload    []byte
}

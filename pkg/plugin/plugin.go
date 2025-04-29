package plugin

import (
	"encoding/json"
	"net/http"
)

type Plugin interface {
	ExecuteOutbound(outbound *Outbound, context *Context) error
	ExecuteInbound(inbound *Inbound) (InboundResult, error)
	ValidateConfig() error
	MarshalConfig() ([]byte, error)
}

type BasePlugin[T any] struct {
	Name   string
	Config T
}

func (p *BasePlugin[T]) UnmarshalConfig(data []byte) error {
	return json.Unmarshal(data, &p.Config)
}

func (p *BasePlugin[T]) MarshalConfig() ([]byte, error) {
	return json.Marshal(p.Config)
}

func (p *BasePlugin[T]) ExecuteOutbound(outbound *Outbound, context *Context) error {
	panic("not implemented")
}

func (p *BasePlugin[T]) ExecuteInbound(inbound *Inbound) (InboundResult, error) {
	panic("not implemented")
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

type Context struct {
	//Workspace *entities.Workspace
}

type InboundResult struct {
	Terminated bool
	Payload    []byte
}

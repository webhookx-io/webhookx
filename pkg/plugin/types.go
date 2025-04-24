package plugin

import (
	"encoding/json"
	"net/http"
)

type Plugin interface {
	ExecuteOutbound(r *OutboundRequest, context *Context) error
	ExecuteInbound(r *http.Request, body []byte, w http.ResponseWriter) (InboundResult, error)
	ValidateConfig() error
	MarshalConfig() ([]byte, error)
}

type OutboundRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Payload string            `json:"payload"`
}

type Context struct {
	//Workspace *entities.Workspace
}

type InboundResult struct {
	Terminated bool
	Payload    []byte
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

func (p *BasePlugin[T]) ExecuteOutbound(r *OutboundRequest, context *Context) error {
	panic("not implemented")
}

func (p *BasePlugin[T]) ExecuteInbound(r *http.Request, body []byte, w http.ResponseWriter) (InboundResult, error) {
	panic("not implemented")
}

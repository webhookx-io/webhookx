package plugin

import (
	"encoding/json"
	"net/http"
)

type Plugin interface {
	ExecuteOutbound(req *Request, context *Context) error
	ExecuteInbound(r *http.Request, w http.ResponseWriter) error
	ValidateConfig() error
	MarshalConfig() ([]byte, error)
}

type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Payload string            `json:"payload"`
}

type Context struct {
	//Workspace *entities.Workspace
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

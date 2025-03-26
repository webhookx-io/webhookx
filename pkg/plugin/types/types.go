package types

import "github.com/webhookx-io/webhookx/db/entities"

type PluginConfig interface {
	Validate() error
	ProcessDefault()
}

type Plugin interface {
	Execute(req *Request, context *Context) error
	Config() PluginConfig
}

type Request struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Payload string            `json:"payload"`
}

type Context struct {
	Workspace *entities.Workspace
}

type BasePlugin struct {
	Name string
}

package types

import "github.com/webhookx-io/webhookx/db/entities"

type PluginConfig interface {
	Validate() error
	ProcessDefault()
}

type Plugin interface {
	Execute(req *Request, context *Context)
	Config() PluginConfig
	GetName() string
}

type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Payload []byte
}

type Context struct {
	Workspace *entities.Workspace
}

type BasePlugin struct {
	Name string
}

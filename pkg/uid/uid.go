package uid

import (
	"go.jetify.com/typeid/v2"
)

type Prefix string

func (p Prefix) String() string {
	return string(p)
}

const (
	AttemptPrefix   Prefix = "at"
	EndpointPrefix  Prefix = "end"
	EventPrefix     Prefix = "evt"
	PluginPrefix    Prefix = "plg"
	SourcePrefix    Prefix = "src"
	WorkspacePrefix Prefix = "ws"
)

func Generate(prefix Prefix) string {
	return genTypeID(prefix.String())
}


func genTypeID(prefix string) string {
	return typeid.MustGenerate(prefix).String()
}

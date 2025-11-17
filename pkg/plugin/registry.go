package plugin

import (
	"fmt"
	"sync"
)

type Type string

const (
	TypeInbound  Type = "inbound"
	TypeOutbound Type = "outbound"
)

type Registration struct {
	Type    Type
	Factory func() Plugin
}

var mux sync.RWMutex
var registry = map[string]*Registration{}

func RegisterPlugin(typ Type, name string, fn func() Plugin) {
	mux.Lock()
	defer mux.Unlock()
	if _, ok := registry[name]; ok {
		panic(fmt.Sprintf("plugin '%s' already registered", name))
	}

	registry[name] = &Registration{
		Type:    typ,
		Factory: fn,
	}
}

func GetRegistration(name string) *Registration {
	mux.RLock()
	defer mux.RUnlock()
	return registry[name]
}

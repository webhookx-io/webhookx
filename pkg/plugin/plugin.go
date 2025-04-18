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

type NewPluginFunc func(config []byte) (Plugin, error)

type Registration struct {
	Type Type
	New  NewPluginFunc
}

var mux sync.Mutex
var registry = make(map[string]*Registration)

func RegisterPlugin(typ Type, name string, fn NewPluginFunc) {
	mux.Lock()
	defer mux.Unlock()
	if _, ok := registry[name]; ok {
		panic(fmt.Sprintf("plugin '%s' already registered", name))
	}

	registry[name] = &Registration{
		Type: typ,
		New:  fn,
	}
}

func GetRegistration(name string) *Registration {
	return registry[name]
}

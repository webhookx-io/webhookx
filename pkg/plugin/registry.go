package plugin

import (
	"fmt"
	"sort"
	"sync"
)

type Type string

const (
	TypeInbound  Type = "inbound"
	TypeOutbound Type = "outbound"
)

type Registration struct {
	Name    string
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
		Name:    name,
		Type:    typ,
		Factory: fn,
	}
}

func GetRegistration(name string) *Registration {
	mux.RLock()
	defer mux.RUnlock()
	return registry[name]
}

func ListRegistration() []Registration {
	mux.RLock()
	registrations := make([]Registration, 0, len(registry))
	for _, registration := range registry {
		registrations = append(registrations, *registration)
	}
	mux.RUnlock()

	sort.Slice(registrations, func(i, j int) bool {
		return registrations[i].Name < registrations[j].Name
	})
	return registrations
}

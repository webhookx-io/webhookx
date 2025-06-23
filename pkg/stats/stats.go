package stats

import (
	"maps"
	"sync"
	"time"
)

type Provider interface {
	Stats() map[string]interface{}
}

type ProviderFunc func() map[string]interface{}

func (f ProviderFunc) Stats() map[string]interface{} {
	return f()
}

var (
	mux       sync.RWMutex
	providers []Provider
)

func Register(p Provider) {
	mux.Lock()
	defer mux.Unlock()
	providers = append(providers, p)
}

type Stats map[string]interface{}

func (m Stats) Int(key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return v.(int)
}

func (m Stats) Int64(key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return v.(int64)
}

func (m Stats) Time(key string) time.Time {
	v, ok := m[key]
	if !ok {
		return time.Time{}
	}
	return v.(time.Time)
}

func Collect() Stats {
	mux.RLock()
	defer mux.RUnlock()

	stats := make(map[string]interface{})
	for _, p := range providers {
		maps.Copy(stats, p.Stats())
	}
	return stats
}

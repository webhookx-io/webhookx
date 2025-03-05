package store

import "sync"

var mux sync.RWMutex
var store = map[string]any{}

// Get looks up a key's value
func Get(key string) (any, bool) {
	mux.RLock()
	defer mux.RUnlock()
	if v, ok := store[key]; ok {
		return v, true
	}
	return nil, false
}

// GetDefault looks up a key's value, returns def if not exist
func GetDefault(key string, def any) any {
	v, ok := Get(key)
	if !ok {
		return def
	}
	return v
}

// Set sets the key-value entry
func Set(key string, value interface{}) {
	mux.Lock()
	defer mux.Unlock()
	store[key] = value
}

// Remove removes the key's entry
func Remove(key string) {
	mux.Lock()
	defer mux.Unlock()
	delete(store, key)
}

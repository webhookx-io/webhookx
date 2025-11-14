package constants

import "strings"

// CacheKey cache key definition.
// format "webhookx:<name>:<version>:<id>"
type CacheKey struct {
	Name    string
	Version string
}

func (c CacheKey) Build(id string) string {
	var sb strings.Builder
	sb.WriteString("webhookx:")
	sb.WriteString(c.Name)
	sb.WriteString(":")
	sb.WriteString(c.Version)
	sb.WriteString(":")
	sb.WriteString(id)
	return sb.String()
}

var (
	EventCacheKey         = register(CacheKey{"events", "v1"})
	EndpointCacheKey      = register(CacheKey{"endpoints", "v1"})
	EndpointPluginsKey    = register(CacheKey{"endpoint_plugins", "v1"})
	SourcePluginsKey      = register(CacheKey{"source_plugins", "v1"})
	SourceCacheKey        = register(CacheKey{"sources", "v1"})
	WorkspaceCacheKey     = register(CacheKey{"workspaces", "v1"})
	AttemptCacheKey       = register(CacheKey{"attempts", "v1"})
	PluginCacheKey        = register(CacheKey{"plugins", "v1"})
	AttemptDetailCacheKey = register(CacheKey{"attempt_details", "v1"})
	WorkspaceEndpointsKey = register(CacheKey{"workspaces_endpoints", "v1"})
)

var registry = map[string]CacheKey{}

func register(ck CacheKey) CacheKey {
	registry[ck.Name] = ck
	return ck
}

func CacheKeyFrom(name string) CacheKey {
	return registry[name]
}

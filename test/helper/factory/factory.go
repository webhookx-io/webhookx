package factory

import (
	"encoding/json"

	"github.com/creasty/defaults"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/openapi"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
)

func SetDefault(schema *openapi3.Schema, obj interface{}) {
	def := map[string]interface{}{}
	if err := openapi.SetDefaults(schema, def); err != nil {
		panic(err)
	}
	defJSON, err := json.Marshal(def)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(defJSON, &obj); err != nil {
		panic(err)
	}
}

// Endpoint

func defaultEndpoint() entities.Endpoint {
	var entity entities.Endpoint
	SetDefault(entities.LookupSchema("Endpoint"), &entity)

	entity.ID = utils.KSUID()
	entity.Request = entities.RequestConfig{
		URL:    "http://localhost:9999/anything",
		Method: "POST",
	}
	entity.Retry.Config.Attempts = []int64{0, 3, 3}
	entity.Events = []string{"foo.bar"}

	return entity
}

type EndpointOption func(*entities.Endpoint)

func Endpoint(opts ...EndpointOption) *entities.Endpoint {
	e := defaultEndpoint()
	for _, opt := range opts {
		opt(&e)
	}
	return &e
}

func EndpointWS(wid string, opts ...EndpointOption) *entities.Endpoint {
	p := Endpoint(opts...)
	p.WorkspaceId = wid
	return p
}

func WithEndpointPlugins(plugins ...*entities.Plugin) EndpointOption {
	return func(o *entities.Endpoint) {
		o.Plugins = append(o.Plugins, plugins...)
	}
}

// Source

func defaultSource() entities.Source {
	var entity entities.Source
	SetDefault(entities.LookupSchema("Source"), &entity)

	entity.ID = utils.KSUID()
	entity.Type = "http"
	entity.Config.HTTP.Path = "/"
	entity.Config.HTTP.Methods = []string{"POST"}

	return entity
}

type SourceOption func(*entities.Source)

func Source(opts ...SourceOption) *entities.Source {
	e := defaultSource()
	for _, opt := range opts {
		opt(&e)
	}
	return &e
}

func SourceWS(wid string, opts ...SourceOption) *entities.Source {
	p := Source(opts...)
	p.WorkspaceId = wid
	return p
}

func WithSourcePlugins(plugins ...*entities.Plugin) SourceOption {
	return func(o *entities.Source) {
		o.Plugins = append(o.Plugins, plugins...)
	}
}

// Plugin

func defaultPlugin() entities.Plugin {
	var entity entities.Plugin
	SetDefault(entities.LookupSchema("Plugin"), &entity)

	entity.ID = utils.KSUID()
	entity.Config = make(map[string]interface{})

	return entity
}

type PluginOption func(*entities.Plugin)

func WithPluginConfig(config plugin.Configuration) PluginOption {
	return func(e *entities.Plugin) {
		properties, err := utils.StructToMap(config)
		if err != nil {
			panic(err)
		}
		e.Config = properties
	}
}

func Plugin(name string, opts ...PluginOption) *entities.Plugin {
	e := defaultPlugin()
	e.Name = name
	for _, opt := range opts {
		opt(&e)
	}
	return &e
}

func PluginWS(name string, wid string, opts ...PluginOption) *entities.Plugin {
	p := Plugin(name, opts...)
	p.WorkspaceId = wid
	return p
}

// Event

func defaultEvent() entities.Event {
	var entity entities.Event
	defaults.Set(&entity)

	entity.ID = utils.KSUID()
	entity.EventType = "foo.bar"
	entity.Data = []byte("{}")

	return entity
}

type EventOption func(*entities.Event)

func Event(opts ...EventOption) *entities.Event {
	e := defaultEvent()
	for _, opt := range opts {
		opt(&e)
	}
	return &e
}

func EventWS(wid string, opts ...EventOption) *entities.Event {
	p := Event(opts...)
	p.WorkspaceId = wid
	return p
}

// Workspace

func defaultWorkspace() entities.Workspace {
	var entity entities.Workspace
	SetDefault(entities.LookupSchema("Workspace"), &entity)

	entity.ID = utils.KSUID()

	return entity
}

func Workspace(name string) *entities.Workspace {
	e := defaultWorkspace()
	e.Name = &name
	return &e
}

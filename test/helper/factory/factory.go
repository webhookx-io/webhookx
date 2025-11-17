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

func WithEndpointID(id string) EndpointOption {
	return func(e *entities.Endpoint) {
		e.ID = id
	}
}

func WithEndpointName(name string) EndpointOption {
	return func(e *entities.Endpoint) {
		e.Name = &name
	}
}

func WithEndpointMetadata(metadata map[string]string) EndpointOption {
	return func(e *entities.Endpoint) {
		e.Metadata = metadata
	}
}

func Endpoint(opts ...EndpointOption) entities.Endpoint {
	e := defaultEndpoint()
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func EndpointP(opts ...EndpointOption) *entities.Endpoint {
	e := Endpoint(opts...)
	return &e
}

func EndpointWS(wid string, opts ...EndpointOption) entities.Endpoint {
	p := Endpoint(opts...)
	p.WorkspaceId = wid
	return p
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

func WithSourceID(id string) SourceOption {
	return func(e *entities.Source) {
		e.ID = id
	}
}

func WithSourceAsync(async bool) SourceOption {
	return func(e *entities.Source) {
		e.Async = async
	}
}

func WithSourceMetadata(metadata map[string]string) SourceOption {
	return func(e *entities.Source) {
		e.Metadata = metadata
	}
}

func Source(opts ...SourceOption) entities.Source {
	e := defaultSource()
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func SourceP(opts ...SourceOption) *entities.Source {
	e := Source(opts...)
	return &e
}

func SourceWS(wid string, opts ...SourceOption) entities.Source {
	p := Source(opts...)
	p.WorkspaceId = wid
	return p
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

func WithPluginID(id string) PluginOption {
	return func(e *entities.Plugin) {
		e.ID = id
	}
}

func WithPluginEndpointID(endpointID string) PluginOption {
	return func(e *entities.Plugin) {
		e.EndpointId = &endpointID
	}
}

func WithPluginSourceID(sourceID string) PluginOption {
	return func(e *entities.Plugin) {
		e.SourceId = &sourceID
	}
}

func WithPluginName(name string) PluginOption {
	return func(e *entities.Plugin) {
		e.Name = name
	}
}

func WithPluginMetadata(metadata map[string]string) PluginOption {
	return func(e *entities.Plugin) {
		e.Metadata = metadata
	}
}

func WithPluginConfig(config plugin.Configuration) PluginOption {
	return func(e *entities.Plugin) {
		properties, err := utils.StructToMap(config)
		if err != nil {
			panic(err)
		}
		e.Config = properties
	}
}

func Plugin(opts ...PluginOption) entities.Plugin {
	e := defaultPlugin()
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func PluginP(opts ...PluginOption) *entities.Plugin {
	e := Plugin(opts...)
	return &e
}

func PluginWS(wid string, opts ...PluginOption) entities.Plugin {
	p := Plugin(opts...)
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

func Event(opts ...EventOption) entities.Event {
	e := defaultEvent()
	for _, opt := range opts {
		opt(&e)
	}
	return e
}

func EventP(opts ...EventOption) *entities.Event {
	e := Event(opts...)
	return &e
}

func EventWS(wid string, opts ...EventOption) entities.Event {
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

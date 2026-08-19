package api

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Catalog struct {
	Plugins []CatalogPlugin `json:"plugins"`
}

type CatalogPlugin struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	Description string           `json:"description"`
	Schema      *openapi3.Schema `json:"schema"`
}

func (api *API) GetCatalog(w http.ResponseWriter, _ *http.Request) {
	registrations := plugin.ListRegistration()
	plugins := make([]CatalogPlugin, 0, len(registrations))

	for _, registration := range registrations {
		p := registration.Factory()
		plugins = append(plugins, CatalogPlugin{
			Name:        registration.Name,
			Type:        string(registration.Type),
			Description: p.Description(),
			Schema:      p.ConfigSchema(),
		})
	}

	catalog := &Catalog{
		Plugins: plugins,
	}

	api.json(http.StatusOK, w, catalog)
}

package declarative

import (
	"context"

	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/db/query"
	"github.com/webhookx-io/webhookx/utils"
)

type Declarative struct {
	db *db.DB
}

func NewDeclarative(db *db.DB) *Declarative {
	return &Declarative{
		db: db,
	}
}

func (m *Declarative) Sync(wid string, cfg *Configuration) error {
	ctx := context.Background()

	upsertedEndpoints := make(map[string]bool)
	upsertedSources := make(map[string]bool)

	err := m.db.TX(ctx, func(ctx context.Context) error {
		// Endpoints
		for _, endpoint := range cfg.Endpoints {
			endpoint.WorkspaceId = wid
			err := m.db.Endpoints.Upsert(ctx, []string{"ws_id", "name"}, &endpoint.Endpoint)
			if err != nil {
				return err
			}
			upsertedEndpoints[endpoint.ID] = true

			// Plugins
			for _, model := range endpoint.Plugins {
				model.WorkspaceId = wid
				model.EndpointId = utils.Pointer(endpoint.ID)
				err = m.db.Plugins.Upsert(ctx, []string{"endpoint_id", "name"}, model)
				if err != nil {
					return err
				}
			}
		}

		// Sources
		for _, source := range cfg.Sources {
			source.WorkspaceId = wid
			err := m.db.Sources.Upsert(ctx, []string{"ws_id", "name"}, &source.Source)
			if err != nil {
				return err
			}
			upsertedSources[source.ID] = true

			// Plugins
			for _, model := range source.Plugins {
				model.WorkspaceId = wid
				model.SourceId = utils.Pointer(source.ID)
				err = m.db.Plugins.Upsert(ctx, []string{"source_id", "name"}, model)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// cleanup existing endpoint
	var endpointQ query.EndpointQuery
	endpointQ.WorkspaceId = &wid
	endpoints, err := m.db.Endpoints.List(ctx, &endpointQ)
	if err != nil {
		return err
	}
	for _, endpoint := range endpoints {
		if !upsertedEndpoints[endpoint.ID] {
			_, err := m.db.Endpoints.Delete(ctx, endpoint.ID)
			if err != nil {
				return err
			}
		}
	}

	// cleanup existing source
	var sourceQ query.SourceQuery
	sourceQ.WorkspaceId = &wid
	sources, err := m.db.Sources.List(ctx, &sourceQ)
	if err != nil {
		return err
	}
	for _, source := range sources {
		if !upsertedSources[source.ID] {
			_, err := m.db.Sources.Delete(ctx, source.ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Declarative) Dump(ctx context.Context, wid string) (*Configuration, error) {
	var cfg Configuration

	err := m.db.TX(ctx, func(ctx context.Context) error {
		var endpointQ query.EndpointQuery
		endpointQ.WorkspaceId = &wid
		endpoints, err := m.db.Endpoints.List(ctx, &endpointQ)
		if err != nil {
			return err
		}
		for _, endpoint := range endpoints {
			var e Endpoint
			e.Endpoint = *endpoint
			var q query.PluginQuery
			q.EndpointId = &endpoint.ID
			q.WorkspaceId = &endpoint.WorkspaceId
			plugins, err := m.db.Plugins.List(ctx, &q)
			if err != nil {
				return err
			}
			e.Plugins = plugins
			cfg.Endpoints = append(cfg.Endpoints, &e)
		}

		var sourceQ query.SourceQuery
		sourceQ.WorkspaceId = &wid
		sources, err := m.db.Sources.List(ctx, &sourceQ)
		if err != nil {
			return err
		}
		for _, source := range sources {
			var e Source
			e.Source = *source
			var q query.PluginQuery
			q.SourceId = &source.ID
			q.WorkspaceId = &source.WorkspaceId
			plugins, err := m.db.Plugins.List(ctx, &q)
			if err != nil {
				return err
			}
			e.Plugins = plugins
			cfg.Sources = append(cfg.Sources, &e)
		}

		return nil
	})

	return &cfg, err
}

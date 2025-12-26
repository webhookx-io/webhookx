package declarative

import (
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/utils"
)

// Configuration declarative configuration
type Configuration struct {
	Endpoints []*Endpoint `json:"endpoints"`
	Sources   []*Source   `json:"sources"`
}

func (cfg *Configuration) SchemaName() string {
	return "Configuration"
}

// Init initializes entities
func (cfg *Configuration) Init() {
	for _, m := range cfg.Sources {
		if m.ID == "" {
			m.ID = utils.KSUID()
		}
		for _, p := range m.Plugins {
			if p.ID == "" {
				p.ID = utils.KSUID()
			}
			p.SourceId = utils.Pointer(m.ID)
		}
	}
	for _, m := range cfg.Endpoints {
		if m.ID == "" {
			m.ID = utils.KSUID()
		}
		for _, p := range m.Plugins {
			if p.ID == "" {
				p.ID = utils.KSUID()
			}
			p.EndpointId = utils.Pointer(m.ID)
		}
	}
}

func (cfg *Configuration) Validate() error {
	err := utils.Validate(cfg)
	if err != nil {
		return err
	}

	for _, end := range cfg.Endpoints {
		for _, model := range end.Plugins {
			if err := model.Validate(); err != nil {
				return err
			}
		}
	}

	for _, src := range cfg.Sources {
		for _, model := range src.Plugins {
			if err := model.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

type Endpoint struct {
	entities.Endpoint
	Plugins []*entities.Plugin `json:"plugins"`
}

type Source struct {
	entities.Source
	Plugins []*entities.Plugin `json:"plugins"`
}

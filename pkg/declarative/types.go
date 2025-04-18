package declarative

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/utils"
)

// Configuration declarative configuration
type Configuration struct {
	Endpoints []*Endpoint `json:"endpoints" validate:"dive,required"`
	Sources   []*Source   `json:"sources" validate:"dive,required"`
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
			p, err := model.ToPlugin()
			if err != nil {
				return err
			}
			model.Config = utils.Must(p.MarshalConfig())
		}
	}

	for _, src := range cfg.Sources {
		for _, model := range src.Plugins {
			if err := model.Validate(); err != nil {
				return err
			}
			p, err := model.ToPlugin()
			if err != nil {
				return err
			}
			model.Config = utils.Must(p.MarshalConfig())
		}
	}

	return nil
}

type Endpoint struct {
	entities.Endpoint `yaml:",inline"`
	Plugins           []*entities.Plugin `json:"plugins" validate:"dive,required"`
}

func (m *Endpoint) UnmarshalJSON(data []byte) error {
	err := defaults.Set(m)
	if err != nil {
		return err
	}
	type alias Endpoint
	return json.Unmarshal(data, (*alias)(m))
}

type Source struct {
	entities.Source `yaml:",inline"`
	Plugins         []*entities.Plugin `json:"plugins" validate:"dive,required"`
}

func (m *Source) UnmarshalJSON(data []byte) error {
	err := defaults.Set(m)
	if err != nil {
		return err
	}
	type alias Source
	return json.Unmarshal(data, (*alias)(m))
}

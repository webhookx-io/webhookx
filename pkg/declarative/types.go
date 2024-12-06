package declarative

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/db/entities"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
)

// Configuration declarative configuration
type Configuration struct {
	Version   string             `json:"version"`
	Sources   []*entities.Source `json:"sources"`
	Endpoints []*Endpoint        `json:"endpoints"`
}

func (m *Configuration) Validate() error {
	err := utils.Validate(m)
	if err != nil {
		return err
	}

	for _, end := range m.Endpoints {
		for _, model := range end.Plugins {
			cfg, err := plugin.NewConfiguration(model.Name, string(model.Config))
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			model.Config = utils.Must(json.Marshal(cfg))
		}
	}

	return nil
}

type Endpoint struct {
	entities.Endpoint
	Plugins []*entities.Plugin `json:"plugins"`
}

func (m *Endpoint) UnmarshalJSON(data []byte) error {
	err := defaults.Set(m)
	if err != nil {
		return err
	}
	type alias Endpoint
	return json.Unmarshal(data, (*alias)(m))
}

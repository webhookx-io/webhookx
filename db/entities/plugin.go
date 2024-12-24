package entities

import (
	"encoding/json"
	"errors"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/utils"
)

type Plugin struct {
	ID         string              `json:"id" validate:"required"`
	Name       string              `json:"name" validate:"required"`
	Enabled    bool                `json:"enabled" db:"enabled" default:"true"`
	EndpointId string              `json:"endpoint_id" db:"endpoint_id" validate:"required" yaml:"endpoint_id"`
	Config     PluginConfiguration `json:"config"`

	BaseModel `yaml:"-"`
}

func (m *Plugin) Validate() error {
	return utils.Validate(m)
}

func (m *Plugin) UnmarshalJSON(data []byte) error {
	err := defaults.Set(m)
	if err != nil {
		return err
	}
	type alias Plugin
	return json.Unmarshal(data, (*alias)(m))
}

func (m *Plugin) Init() {
	m.ID = utils.KSUID()
	m.Enabled = true
}

type PluginConfiguration json.RawMessage

func (m PluginConfiguration) MarshalYAML() (interface{}, error) {
	if len(m) == 0 {
		return nil, nil
	}
	data := make(map[string]interface{})
	err := json.Unmarshal(m, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m PluginConfiguration) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

func (m *PluginConfiguration) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("json.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

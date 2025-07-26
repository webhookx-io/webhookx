package entities

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Plugin struct {
	ID         string              `json:"id" db:"id"`
	Name       string              `json:"name" db:"name"`
	Enabled    bool                `json:"enabled" db:"enabled"`
	EndpointId *string             `json:"endpoint_id" db:"endpoint_id" yaml:"endpoint_id"`
	SourceId   *string             `json:"source_id" db:"source_id" yaml:"source_id"`
	Config     PluginConfiguration `json:"config" db:"config"`
	Metadata   Metadata            `json:"metadata" db:"metadata"`

	BaseModel `yaml:"-"`
}

func (m *Plugin) Validate() error {
	r := plugin.GetRegistration(m.Name)
	if r == nil {
		e := errs.NewValidateError(errors.New("request validation"))
		e.Fields["name"] = fmt.Sprintf("unknown plugin name '%s'", m.Name)
		return e
	}
	if r.Type == plugin.TypeInbound && m.SourceId == nil {
		e := errs.NewValidateError(errors.New("request validation"))
		e.Fields["source_id"] = fmt.Sprintf("source_id is required for plugin '%s'", m.Name)
		return e
	}
	if r.Type == plugin.TypeOutbound && m.EndpointId == nil {
		e := errs.NewValidateError(errors.New("request validation"))
		e.Fields["endpoint_id"] = fmt.Sprintf("endpoint_id is required for plugin '%s'", m.Name)
		return e
	}

	// validate plugin configuration
	p, err := m.Plugin()
	if err != nil {
		return err
	}
	if err = p.ValidateConfig(); err != nil {
		if e, ok := err.(*errs.ValidateError); ok {
			e.Fields = map[string]interface{}{
				"config": e.Fields,
			}
			return e
		}
		return err
	}
	return nil
}

func (m *Plugin) UnmarshalJSON(data []byte) error {
	err := defaults.Set(m)
	if err != nil {
		return err
	}
	type alias Plugin
	return json.Unmarshal(data, (*alias)(m))
}

func (m *Plugin) Plugin() (plugin.Plugin, error) {
	r := plugin.GetRegistration(m.Name)
	if r == nil {
		return nil, fmt.Errorf("unknown plugin name: '%s'", m.Name)
	}

	executor, err := r.New(m.Config)
	if err != nil {
		return nil, err
	}
	return executor, nil
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

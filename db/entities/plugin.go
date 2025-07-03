package entities

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creasty/defaults"
	"github.com/go-playground/validator/v10"
	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/plugin"
	"github.com/webhookx-io/webhookx/utils"
)

type Plugin struct {
	ID         string              `json:"id" db:"id"`
	Name       string              `json:"name" db:"name" validate:"required,plugin-name"`
	Enabled    bool                `json:"enabled" db:"enabled" default:"true"`
	EndpointId *string             `json:"endpoint_id" db:"endpoint_id" yaml:"endpoint_id"`
	SourceId   *string             `json:"source_id" db:"source_id" yaml:"source_id"`
	Config     PluginConfiguration `json:"config" db:"config"`
	Metadata   Metadata            `json:"metadata" db:"metadata"`

	BaseModel `yaml:"-"`
}

func init() {
	utils.RegisterValidation("plugin-name", func(fl validator.FieldLevel) bool {
		return plugin.GetRegistration(fl.Field().String()) != nil
	})
	utils.RegisterFormatter("plugin-name", func(fe validator.FieldError) string {
		return fmt.Sprintf("unknown plugin name '%s'", fe.Value())
	})
}

func (m *Plugin) Validate() error {
	if err := utils.Validate(m); err != nil {
		return err
	}
	r := plugin.GetRegistration(m.Name)
	if r == nil {
		return fmt.Errorf("unknown plugin name: '%s'", m.Name)
	}
	if r.Type == plugin.TypeInbound && m.SourceId == nil {
		return fmt.Errorf("source_id is required for plugin '%s'", m.Name)
	}
	if r.Type == plugin.TypeOutbound && m.EndpointId == nil {
		return fmt.Errorf("endpoint_id is required for plugin '%s'", m.Name)
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

func (m *Plugin) Init() {
	m.ID = utils.KSUID()
	m.Enabled = true
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

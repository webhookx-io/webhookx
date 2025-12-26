package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/webhookx-io/webhookx/pkg/errs"
	"github.com/webhookx-io/webhookx/pkg/license"
	"github.com/webhookx-io/webhookx/pkg/plugin"
)

type Plugin struct {
	ID         string              `json:"id" db:"id"`
	Name       string              `json:"name" db:"name"`
	Enabled    bool                `json:"enabled" db:"enabled"`
	EndpointId *string             `json:"endpoint_id" db:"endpoint_id"`
	SourceId   *string             `json:"source_id" db:"source_id"`
	Config     PluginConfiguration `json:"config" db:"config"`
	Metadata   Metadata            `json:"metadata" db:"metadata"`

	BaseModel
}

func (m *Plugin) SchemaName() string {
	return "Plugin"
}

func (m *Plugin) Validate() error {
	r := plugin.GetRegistration(m.Name)
	if r == nil {
		e := errs.NewValidateError(errors.New("request validation"))
		e.Fields["name"] = fmt.Sprintf("unknown plugin name '%s'", m.Name)
		return e
	}

	if !license.GetLicenser().AllowPlugin(m.Name) {
		return errs.NewLicenseError(fmt.Errorf("plugin '%s' is not available for current license", m.Name))
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
	p, err := m.ToPlugin()
	if err != nil {
		return err
	}
	if err = p.ValidateConfig(m.Config); err != nil {
		if e, ok := err.(*errs.ValidateError); ok {
			e.Fields = map[string]interface{}{
				"config": e.Fields,
			}
			return e
		}
		return err
	}

	err = p.Init(m.Config)
	if err != nil {
		return err
	}
	m.Config = p.GetConfig()

	return nil
}

func (m *Plugin) ToPlugin() (plugin.Plugin, error) {
	executor, ok := plugin.New(m.Name)
	if !ok {
		return nil, fmt.Errorf("unknown plugin name: '%s'", m.Name)
	}
	return executor, nil
}

type PluginConfiguration map[string]interface{}

func (m *PluginConfiguration) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m PluginConfiguration) Value() (driver.Value, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

func (m *PluginConfiguration) UnmarshalJSON(data []byte) error {
	v := make(map[string]interface{})
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*m = v
	return nil
}

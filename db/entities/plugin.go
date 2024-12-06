package entities

import (
	"encoding/json"
	"github.com/creasty/defaults"
	"github.com/webhookx-io/webhookx/utils"
)

type Plugin struct {
	ID         string          `json:"id" validate:"required"`
	Name       string          `json:"name" validate:"required"`
	Enabled    bool            `json:"enabled" db:"enabled" default:"true"`
	EndpointId string          `json:"endpoint_id" db:"endpoint_id" validate:"required"`
	Config     json.RawMessage `json:"config"`

	BaseModel
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

package entities

import (
	"encoding/json"
	"github.com/webhookx-io/webhookx/pkg/types"
	"github.com/webhookx-io/webhookx/utils"
)

type Event struct {
	ID         string          `json:"id" validate:"required"`
	EventType  string          `json:"event_type" db:"event_type" validate:"required"`
	Data       json.RawMessage `json:"data" validate:"required"`
	IngestedAt types.Time      `json:"ingested_at" db:"ingested_at"`

	BaseModel
}

func (m *Event) SchemaName() string {
	return "Event"
}

func (m *Event) Validate() error {
	return utils.Validate(m)
}

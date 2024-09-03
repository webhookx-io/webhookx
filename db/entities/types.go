package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/webhookx-io/webhookx/pkg/types"
)

type Metadata map[string]string

func (m *Metadata) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m *Metadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type BaseModel struct {
	CreatedAt   types.UnixTime `db:"created_at" json:"created_at"`
	UpdatedAt   types.UnixTime `db:"updated_at" json:"updated_at"`
	WorkspaceId string         `db:"ws_id" json:"-"`
}

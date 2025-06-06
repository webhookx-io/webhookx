package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/lib/pq"
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
	CreatedAt   types.Time `db:"created_at" json:"created_at"`
	UpdatedAt   types.Time `db:"updated_at" json:"updated_at"`
	WorkspaceId string     `db:"ws_id" json:"-"`
}

type Headers map[string]string

func (m *Headers) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		return json.Unmarshal([]byte(v), m)
	case []byte:
		return json.Unmarshal(v, m)
	default:
		return errors.New("unknown type")
	}
}

func (m Headers) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type Strings = pq.StringArray

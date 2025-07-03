package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/webhookx-io/webhookx/pkg/types"
)

type Metadata struct {
	items map[string]string
}

func (m *Metadata) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), &m.items)
}

func (m Metadata) Value() (driver.Value, error) {
	return m.MarshalJSON()
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	m.items = make(map[string]string)
	if err := json.Unmarshal(data, &m.items); err != nil {
		return err
	}
	return nil
}

func (m Metadata) MarshalJSON() ([]byte, error) {
	if m.items == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m.items)
}

type BaseModel struct {
	CreatedAt   types.Time `db:"created_at" json:"created_at"`
	UpdatedAt   types.Time `db:"updated_at" json:"updated_at"`
	WorkspaceId string     `db:"ws_id" json:"-"`
}

type Headers map[string]string

func (m *Headers) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m Headers) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type Strings = pq.StringArray

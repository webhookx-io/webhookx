package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/webhookx-io/webhookx/pkg/types"
)

type Metadata map[string]string

func (m *Metadata) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(m)
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	v := make(map[string]string)
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*m = v
	return nil
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

func (m *Headers) UnmarshalJSON(data []byte) error {
	v := make(map[string]string)
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*m = v
	return nil
}

type Strings = pq.StringArray

type RateLimit struct {
	Quota  int `json:"quota"`
	Period int `json:"period"`
}

func (m *RateLimit) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m RateLimit) Value() (driver.Value, error) {
	return json.Marshal(m)
}

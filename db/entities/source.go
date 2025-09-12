package entities

import (
	"database/sql/driver"
	"encoding/json"
)

type CustomResponse struct {
	Code        int    `json:"code"`
	ContentType string `json:"content_type" yaml:"content_type"`
	Body        string `json:"body"`
}

func (m *CustomResponse) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m CustomResponse) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type Source struct {
	ID        string          `json:"id" db:"id"`
	Name      *string         `json:"name" db:"name"`
	Enabled   bool            `json:"enabled" db:"enabled"`
	Path      string          `json:"path" db:"path"`
	Methods   Strings         `json:"methods" db:"methods"`
	Async     bool            `json:"async" db:"async"`
	Response  *CustomResponse `json:"response" db:"response"`
	Metadata  Metadata        `json:"metadata" db:"metadata"`
	RateLimit *RateLimit      `json:"rate_limit" yaml:"rate_limit" db:"rate_limit"`

	BaseModel `yaml:"-"`
}

func (m *Source) SchemaName() string {
	return "Source"
}

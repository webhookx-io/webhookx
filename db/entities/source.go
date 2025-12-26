package entities

import (
	"database/sql/driver"
	"encoding/json"
)

type CustomResponse struct {
	Code        int    `json:"code"`
	ContentType string `json:"content_type"`
	Body        string `json:"body"`
}

func (m *CustomResponse) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m CustomResponse) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type SourceConfig struct {
	HTTP HttpSourceConfig `json:"http"`
}

func (m *SourceConfig) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m SourceConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type HttpSourceConfig struct {
	Path     string          `json:"path"`
	Methods  Strings         `json:"methods"`
	Response *CustomResponse `json:"response"`
}

type Source struct {
	ID        string       `json:"id" db:"id"`
	Name      *string      `json:"name" db:"name"`
	Enabled   bool         `json:"enabled" db:"enabled"`
	Type      string       `json:"type" db:"type"`
	Config    SourceConfig `json:"config" db:"config"`
	Async     bool         `json:"async" db:"async"`
	Metadata  Metadata     `json:"metadata" db:"metadata"`
	RateLimit *RateLimit   `json:"rate_limit" db:"rate_limit"`

	Plugins []*Plugin `json:"-" db:"-"`

	BaseModel
}

func (m *Source) SchemaName() string {
	return "Source"
}

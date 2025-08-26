package entities

import (
	"database/sql/driver"
	"encoding/json"
)

type Endpoint struct {
	ID          string        `json:"id" db:"id"`
	Name        *string       `json:"name" db:"name"`
	Description *string       `json:"description" db:"description"`
	Enabled     bool          `json:"enabled" db:"enabled"`
	Request     RequestConfig `json:"request" db:"request"`
	Retry       Retry         `json:"retry" db:"retry"`
	Events      Strings       `json:"events" db:"events"`
	Metadata    Metadata      `json:"metadata" db:"metadata"`
	RateLimit   *RateLimit    `json:"rate_limit" yaml:"rate_limit" db:"rate_limit"`

	BaseModel `yaml:"-"`
}

func (m *Endpoint) SchemaName() string {
	return "Endpoint"
}

type RequestConfig struct {
	URL     string  `json:"url"`
	Method  string  `json:"method"`
	Headers Headers `json:"headers"`
	Timeout int64   `json:"timeout"`
}

func (m *RequestConfig) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m RequestConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type RetryStrategy string

const (
	RetryStrategyFixed RetryStrategy = "fixed"
)

func (m RetryStrategy) String() string {
	return string(m)
}

type Retry struct {
	Strategy RetryStrategy       `json:"strategy"`
	Config   FixedStrategyConfig `json:"config"`
}

func (m *Retry) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m Retry) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type FixedStrategyConfig struct {
	Attempts []int64 `json:"attempts"`
}

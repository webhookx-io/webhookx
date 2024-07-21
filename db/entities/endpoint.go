package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/webhookx-io/webhookx/utils"
)

type Endpoint struct {
	ID          string         `json:"id" db:"id"`
	Name        *string        `json:"name" db:"name"`
	Description *string        `json:"description" db:"description"`
	Enabled     bool           `json:"enabled" db:"enabled" default:"true"`
	Request     RequestConfig  `json:"request" db:"request"`
	Retry       Retry          `json:"retry" db:"retry"`
	Events      pq.StringArray `json:"events" db:"events"`

	BaseModel
}

func (m *Endpoint) Init() {
	m.ID = utils.UUID()
	m.Enabled = true
}

func (m *Endpoint) Validate() error {
	return utils.Validate(m)
}

type RequestConfig struct {
	URL     string        `json:"url" validate:"required"`
	Method  string        `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH"`
	Headers []HeaderEntry `json:"headers"`
	Timeout Timeout       `json:"timeout"`
}

func (m *RequestConfig) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m RequestConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type HeaderEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Timeout struct {
	Connect int `json:"connect" default:"60000" validate:"gt=0"`
	Read    int `json:"read" default:"60000" validate:"gt=0"`
	Write   int `json:"write" default:"60000" validate:"gt=0"`
}

type RetryStrategy string

const (
	FixedStrategy RetryStrategy = "fixed"
)

func (m RetryStrategy) String() string {
	return string(m)
}

type Retry struct {
	Strategy RetryStrategy       `json:"strategy" validate:"oneof=fixed" default:"fixed"`
	Config   FixedStrategyConfig `json:"config"`
}

func (m *Retry) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m Retry) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *Retry) Validate() error {
	return nil
}

type FixedStrategyConfig struct {
	Attempts []int64 `json:"attempts" default:"[0,60,3600]"`
}

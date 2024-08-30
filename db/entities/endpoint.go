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
	m.ID = utils.KSUID()
	m.Enabled = true
}

func (m *Endpoint) Validate() error {
	return utils.Validate(m)
}

type RequestConfig struct {
	URL     string            `json:"url" validate:"required"`
	Method  string            `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH"`
	Headers map[string]string `json:"headers"`
	Timeout int64             `json:"timeout" default:"10000" validate:"gte=0"`
}

func (m *RequestConfig) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m RequestConfig) Value() (driver.Value, error) {
	return json.Marshal(m)
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

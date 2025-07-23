package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/webhookx-io/webhookx/utils"
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

	BaseModel `yaml:"-"`
}

// TODO ???
func NewEndpoint() *Endpoint {
	entity := new(Endpoint)
	entity.ID = utils.KSUID()
	defaults := schemas["Endpoint"].Defaults()
	b, _ := json.Marshal(defaults)
	json.Unmarshal(b, entity)
	return entity
}

func (m *Endpoint) Validate() error {
	v := utils.Must(utils.StructToMap(m))
	return schemas["Endpoint"].Validate(v)
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

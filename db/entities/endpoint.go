package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/webhookx-io/webhookx/utils"
)

type Endpoint struct {
	ID          string        `json:"id" db:"id"`
	Name        *string       `json:"name" db:"name"`
	Description *string       `json:"description" db:"description"`
	Enabled     bool          `json:"enabled" db:"enabled" default:"true"`
	Request     RequestConfig `json:"request" db:"request"`
	Retry       Retry         `json:"retry" db:"retry"`
	Events      Strings       `json:"events" db:"events"`
	Metadata    Metadata      `json:"metadata" db:"metadata" default:"{}"`

	BaseModel `yaml:"-"`
}

func (m *Endpoint) Init() {
	m.ID = utils.KSUID()
	m.Enabled = true
}

func NewEndpoint() *Endpoint {
	entity := new(Endpoint)
	entity.ID = utils.KSUID()
	// New an endpoint with default values set according to the jsonschema definition
	// TODO
	data := make(map[string]interface{})
	schema := schemaRegistry["endpoint"]
	schema.VisitJSON(data,
		openapi3.MultiErrors(),
		openapi3.DisableReadOnlyValidation(),
		openapi3.VisitAsRequest(),
		openapi3.DefaultsSet(func() {}),
	)
	b, _ := json.Marshal(data)
	json.Unmarshal(b, entity)

	return entity
}

func (m *Endpoint) Validate() error {
	schema := schemaRegistry["endpoint"]
	b, _ := json.Marshal(m)
	var generic map[string]interface{}
	json.Unmarshal(b, &generic)
	return schema.VisitJSON(generic,
		openapi3.MultiErrors(),
		openapi3.DisableReadOnlyValidation(),
	)
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
	RetryStrategyFixed RetryStrategy = "fixed"
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

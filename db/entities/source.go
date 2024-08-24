package entities

import (
	"database/sql/driver"
	"encoding/json"
	"github.com/lib/pq"
	"github.com/webhookx-io/webhookx/utils"
)

type CustomResponse struct {
	Code        int    `json:"code" validate:"required,gte=200,lte=599"`
	ContentType string `json:"content_type" validate:"required"`
	Body        string `json:"body"`
}

func (m *CustomResponse) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m CustomResponse) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type Source struct {
	ID   string  `json:"id" db:"id"`
	Name *string `json:"name" db:"name"`

	Enabled bool           `json:"enabled" db:"enabled"`
	Path    string         `json:"path" db:"path"`
	Methods pq.StringArray `json:"methods" db:"methods"`

	Response *CustomResponse `json:"response" db:"response"`

	BaseModel
}

func (m *Source) Validate() error {
	return utils.Validate(m)
}

func (m *Source) Init() {
	m.ID = utils.KSUID()
	m.Enabled = true
}

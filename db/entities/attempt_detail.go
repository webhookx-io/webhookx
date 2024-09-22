package entities

import (
	"database/sql/driver"
	"encoding/json"
)

type AttemptDetail struct {
	ID              string  `json:"id" db:"id"`
	RequestHeaders  Headers `json:"request_headers" db:"request_headers"`
	RequestBody     *string `json:"request_body" db:"request_body"`
	ResponseHeaders Headers `json:"response_headers" db:"response_headers"`
	ResponseBody    *string `json:"response_body" db:"response_body"`

	BaseModel
}

type Headers map[string]string

func (m *Headers) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m Headers) Value() (driver.Value, error) {
	return json.Marshal(m)
}

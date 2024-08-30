package entities

import (
	"database/sql/driver"
	"encoding/json"
)

type Attempt struct {
	ID            string            `json:"id" db:"id"`
	EventId       string            `json:"event_id" db:"event_id"`
	EndpointId    string            `json:"endpoint_id" db:"endpoint_id"`
	Status        AttemptStatus     `json:"status" db:"status"`
	AttemptNumber int               `json:"attempt_number" db:"attempt_number"`
	AttemptAt     *int64            `json:"attempt_at" db:"attempt_at"` // attempted_at?
	ErrorCode     *AttemptErrorCode `json:"error_code" db:"error_code"`
	Request       *AttemptRequest   `json:"request" db:"request"`
	Response      *AttemptResponse  `json:"response" db:"response"`

	BaseModel
}

type AttemptStatus = string

const (
	AttemptStatusInit     AttemptStatus = "INIT"
	AttemptStatusQueued   AttemptStatus = "QUEUED"
	AttemptStatusSuccess  AttemptStatus = "SUCCESSFUL"
	AttemptStatusFailure  AttemptStatus = "FAILED"
	AttemptStatusCanceled AttemptStatus = "CANCELED"
)

type AttemptErrorCode = string

const (
	AttemptErrorCodeTimeout          AttemptErrorCode = "TIMEOUT"
	AttemptErrorCodeUnknown          AttemptErrorCode = "UNKNOWN"
	AttemptErrorCodeEndpointDisabled AttemptErrorCode = "ENDPOINT_DISABLED"
	AttemptErrorCodeEndpointNotFound AttemptErrorCode = "ENDPOINT_NOT_FOUND"
)

type AttemptRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}

func (m *AttemptRequest) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m AttemptRequest) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type AttemptResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    *string           `json:"body"`
}

func (m *AttemptResponse) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m AttemptResponse) Value() (driver.Value, error) {
	return json.Marshal(m)
}

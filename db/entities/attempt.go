package entities

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/webhookx-io/webhookx/pkg/types"
)

type Attempt struct {
	ID            string             `json:"id" db:"id"`
	EventId       string             `json:"event_id" db:"event_id"`
	EndpointId    string             `json:"endpoint_id" db:"endpoint_id"`
	Status        AttemptStatus      `json:"status" db:"status"`
	AttemptNumber int                `json:"attempt_number" db:"attempt_number"`
	ScheduledAt   types.Time         `json:"scheduled_at" db:"scheduled_at"`
	AttemptedAt   *types.Time        `json:"attempted_at" db:"attempted_at"`
	TriggerMode   AttemptTriggerMode `json:"trigger_mode" db:"trigger_mode"`
	Exhausted     bool               `json:"exhausted" db:"exhausted"`

	ErrorCode *AttemptErrorCode `json:"error_code" db:"error_code"`
	Request   *AttemptRequest   `json:"request" db:"request"`
	Response  *AttemptResponse  `json:"response" db:"response"`

	Event *Event `json:"-" db:"-"`

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

type AttemptTriggerMode = string

const (
	AttemptTriggerModeInitial   AttemptTriggerMode = "INITIAL"
	AttemptTriggerModeManual    AttemptTriggerMode = "MANUAL"
	AttemptTriggerModeAutomatic AttemptTriggerMode = "AUTOMATIC"
)

type AttemptRequest struct {
	Method  string  `json:"method"`
	URL     string  `json:"url"`
	Headers Headers `json:"headers"`
	Body    *string `json:"body"`
}

func (m *AttemptRequest) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m AttemptRequest) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type AttemptResponse struct {
	Status  int     `json:"status"`
	Latency int64   `json:"latency"`
	Headers Headers `json:"headers"`
	Body    *string `json:"body"`
}

func (m *AttemptResponse) Scan(src interface{}) error {
	return json.Unmarshal(src.([]byte), m)
}

func (m AttemptResponse) Value() (driver.Value, error) {
	return json.Marshal(m)
}

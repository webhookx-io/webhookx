package eventbus

import "encoding/json"

const (
	EventInvalidation = "invalidation"
)

type EventPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
	Time  int64           `json:"time"`
	Node  string          `json:"node"`
}

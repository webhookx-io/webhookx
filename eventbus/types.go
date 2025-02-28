package eventbus

import "encoding/json"

const (
	EventCRUD = "crud"
)

type EventPayload struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
	Time  int64           `json:"time"`
	Node  string          `json:"node"`
}

type CrudData struct {
	ID       string          `json:"id"`
	CacheKey string          `json:"cache_key"`
	Entity   string          `json:"entity"`
	Data     json.RawMessage `json:"data"`
}

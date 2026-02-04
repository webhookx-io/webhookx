package eventbus

import "encoding/json"

const (
	EventCRUD        = "crud"
	EventEventFanout = "event.fanout"
)

type CrudData struct {
	Entity    string          `json:"entity"`
	ID        string          `json:"id"`
	WID       string          `json:"wid"`
	CacheName string          `json:"cache_name"`
	Data      json.RawMessage `json:"data"`
}

func (m *CrudData) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

type EventFanoutData struct {
	EventId    string   `json:"event_id"`
	AttemptIds []string `json:"attempt_ids"`
}

func (m *EventFanoutData) Marshal() ([]byte, error) {
	return json.Marshal(m)
}


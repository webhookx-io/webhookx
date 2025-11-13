package eventbus

import "encoding/json"

const (
	EventCRUD        = "crud"
	EventEventFanout = "event.fanout"
)

type Bus interface {
	ClusteringBroadcast(event string, data interface{}) error
	ClusteringSubscribe(channel string, fn func(data []byte))
	Broadcast(channel string, data interface{})
	Subscribe(channel string, cb Callback)
}

// Message clustering message
type Message struct {
	Event string          `json:"event"`
	Time  int64           `json:"time"`
	Node  string          `json:"node"`
	Data  json.RawMessage `json:"data"`
}

type Callback func(data interface{})

type CrudData struct {
	Entity    string          `json:"entity"`
	ID        string          `json:"id"`
	WID       string          `json:"wid"`
	CacheName string          `json:"cache_name"`
	Data      json.RawMessage `json:"data"`
}

type EventFanoutData struct {
	EventId    string   `json:"event_id"`
	AttemptIds []string `json:"attempt_ids"`
}

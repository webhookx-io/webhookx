package model

type MessageData struct {
	EventID    string `json:"event_id"`
	EndpointId string `json:"endpoint_id"`
	Delay      int64  `json:"delay"`
	Attempt    int    `json:"attempt"`
}

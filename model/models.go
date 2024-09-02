package model

type MessageData struct {
	EventID    string `json:"event_id"`
	EndpointId string `json:"endpoint_id"`
	Attempt    int    `json:"attempt"`
}

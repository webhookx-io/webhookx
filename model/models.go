package model

type MessageData struct {
	EventID     string `json:"event_id"`
	EndpointId  string `json:"endpoint_id"`
	Time        int64  `json:"time"`
	Delay       int64  `json:"delay"`
	Attempt     int    `json:"attempt"`
	AttemptLeft int    `json:"attempt_left"`
}

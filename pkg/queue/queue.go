package queue

import (
	"encoding/json"
	"time"
)

type Task struct {
	ID   string
	data []byte
	Data interface{}
}

func (t *Task) UnmarshalData(v interface{}) error {
	return json.Unmarshal(t.data, v)
}

type TaskQueue interface {
	Add(task *Task, delay time.Duration) error
	Get() (task *Task, err error)
	Delete(task *Task) error
}

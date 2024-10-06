package taskqueue

import (
	"encoding/json"
	"github.com/webhookx-io/webhookx/utils"
	"time"
)

type TaskMessage struct {
	ID string

	data []byte
	Data interface{}
}

func NewTaskMessage(data interface{}) *TaskMessage {
	task := &TaskMessage{
		ID:   utils.UUID(),
		Data: data,
	}
	return task
}

func (t *TaskMessage) String() string {
	return t.ID + ":" + string(t.data)
}

func (t *TaskMessage) UnmarshalData(v interface{}) error {
	return json.Unmarshal(t.data, v)
}

type TaskQueue interface {
	Add(task *TaskMessage, scheduleAt time.Time) error
	Get() (task *TaskMessage, err error)
	Delete(task *TaskMessage) error
}

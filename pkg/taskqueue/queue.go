package taskqueue

import (
	"context"
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
	Add(ctx context.Context, task *TaskMessage, scheduleAt time.Time) error
	Get(ctx context.Context) (task *TaskMessage, err error)
	Delete(ctx context.Context, task *TaskMessage) error
}

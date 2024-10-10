package taskqueue

import (
	"context"
	"encoding/json"
	"time"
)

type TaskMessage struct {
	ID          string
	ScheduledAt time.Time
	Data        interface{}
	data        []byte
}

func (t *TaskMessage) String() string {
	return t.ID + ":" + string(t.data)
}

func (t *TaskMessage) UnmarshalData(v interface{}) error {
	return json.Unmarshal(t.data, v)
}

func (t *TaskMessage) MarshalData() ([]byte, error) {
	return json.Marshal(t.Data)
}

type GetOptions struct {
	Count int64
}

type TaskQueue interface {
	Add(ctx context.Context, tasks []*TaskMessage) error
	Get(ctx context.Context, opts *GetOptions) (tasks []*TaskMessage, err error)
	Delete(ctx context.Context, task *TaskMessage) error
	Size(ctx context.Context) (int64, error)
}

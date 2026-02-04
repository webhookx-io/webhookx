package schedule

import (
	"time"
)

type Task struct {
	Name         string
	InitialDelay time.Duration
	Interval     time.Duration
	Do           func()
}

type Scheduler interface {
	AddTask(task *Task)
	GetTask(name string) *Task
}

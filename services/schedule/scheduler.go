package schedule

import (
	"context"
	"sync"
	"time"
)

// Scheduler manages scheduled tasks
type Scheduler interface {
	Schedule(Task)
	RunNow(name string)
	Start() error
	Stop(context.Context) error
}

type Task struct {
	Name      string
	Scheduled Schedule
	Run       func(ctx context.Context) error
}

type Schedule interface {
	Next(time.Time) time.Time
}

type IntervalSchedule struct {
	once         sync.Once
	InitialDelay time.Duration
	Interval     time.Duration
}

func NewIntervalSchedule(initialDelay time.Duration, interval time.Duration) *IntervalSchedule {
	return &IntervalSchedule{
		InitialDelay: initialDelay,
		Interval:     interval,
	}
}

func (m *IntervalSchedule) Next(t time.Time) time.Time {
	interval := m.Interval
	m.once.Do(func() { interval = m.InitialDelay })
	return t.Add(interval)
}

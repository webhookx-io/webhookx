package schedule

import (
	"errors"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Task struct {
	id cron.EntryID

	Name         string
	InitialDelay time.Duration
	Interval     time.Duration
	Do           func()
}

type Scheduler interface {
	AddTask(task *Task)
	GetTask(name string) *Task
	Start()
	Stop()
}

var (
	ErrTaskAdded = errors.New("task already added")
)

var _ Scheduler = &DefaultScheduler{}

type IntervalSchedule struct {
	once         sync.Once
	InitialDelay time.Duration
	Interval     time.Duration
}

func (s *IntervalSchedule) Next(t time.Time) time.Time {
	interval := s.Interval
	s.once.Do(func() {
		interval = s.InitialDelay
	})
	return t.Add(interval)
}

type DefaultScheduler struct {
	cron  *cron.Cron
	tasks map[string]*Task
	mux   sync.RWMutex
}

func NewScheduler() Scheduler {
	return &DefaultScheduler{
		cron:  cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger))),
		tasks: make(map[string]*Task),
	}
}

func (s *DefaultScheduler) AddTask(task *Task) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, exists := s.tasks[task.Name]; exists {
		panic(ErrTaskAdded)
	}

	schedule := &IntervalSchedule{
		InitialDelay: task.InitialDelay,
		Interval:     task.Interval,
	}
	entryID := s.cron.Schedule(schedule, cron.FuncJob(task.Do))

	task.id = entryID
	s.tasks[task.Name] = task
}

func (s *DefaultScheduler) GetTask(id string) *Task {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.tasks[id]
}

func (s *DefaultScheduler) Start() {
	s.cron.Start()
}

func (s *DefaultScheduler) Stop() {
	ctx := s.cron.Stop()
	<-ctx.Done()
}

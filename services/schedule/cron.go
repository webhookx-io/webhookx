package schedule

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	ErrTaskAdded = errors.New("task already added")
)

type intervalSchedule struct {
	once         sync.Once
	InitialDelay time.Duration
	Interval     time.Duration
}

func (s *intervalSchedule) Next(t time.Time) time.Time {
	interval := s.Interval
	s.once.Do(func() { interval = s.InitialDelay })
	return t.Add(interval)
}

type cronTask struct {
	id cron.EntryID
	*Task
}

type ScheduleService struct {
	cron  *cron.Cron
	tasks map[string]*cronTask
	mux   sync.RWMutex
}

func NewSchedulerService() *ScheduleService {
	return &ScheduleService{
		cron:  cron.New(cron.WithChain(cron.Recover(cron.DefaultLogger))),
		tasks: make(map[string]*cronTask),
	}
}

func (s *ScheduleService) Name() string {
	return "schedule"
}

func (s *ScheduleService) AddTask(task *Task) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, exists := s.tasks[task.Name]; exists {
		panic(ErrTaskAdded)
	}

	schedule := &intervalSchedule{
		InitialDelay: task.InitialDelay,
		Interval:     task.Interval,
	}
	id := s.cron.Schedule(schedule, cron.FuncJob(task.Do))

	s.tasks[task.Name] = &cronTask{
		id:   id,
		Task: task,
	}
}

func (s *ScheduleService) GetTask(id string) *Task {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if task, exists := s.tasks[id]; exists {
		return task.Task
	}
	return nil
}

func (s *ScheduleService) Start() error {
	s.cron.Start()
	return nil
}

func (s *ScheduleService) Stop(_ context.Context) error {
	ctx := s.cron.Stop()
	<-ctx.Done()
	return nil
}

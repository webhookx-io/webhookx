package schedule

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var (
	ErrTaskAdded    = errors.New("task already added")
	ErrTaskNotFound = errors.New("task not found")
)

// CronScheduler is a scheduler implemented by robfig/cron
type CronScheduler struct {
	log     *zap.SugaredLogger
	cron    *cron.Cron
	entries map[string]cron.EntryID
	mux     sync.RWMutex
}

func NewCronScheduler() *CronScheduler {
	return &CronScheduler{
		log: zap.S().Named("cron"),
		cron: cron.New(
			cron.WithChain(
				cron.Recover(cron.DefaultLogger),
			),
		),
		entries: make(map[string]cron.EntryID),
	}
}

func (s *CronScheduler) Name() string {
	return "schedule"
}

func (s *CronScheduler) Schedule(task Task) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, exists := s.entries[task.Name]; exists {
		panic(ErrTaskAdded)
	}

	wrappedFunc := func() {
		start := time.Now()
		err := task.Run(context.TODO())
		elapsed := time.Since(start)
		if err != nil {
			s.log.Errorw("task failed",
				"task", task.Name,
				"elapsed_ms", elapsed.Milliseconds(),
				zap.Error(err),
			)
			return
		}
		// verbose
		// s.log.Debugw(fmt.Sprintf("task '%s' completed", task.Name), "elapsed_ms", elapsed.Milliseconds())
	}

	id := s.cron.Schedule(task.Scheduled, cron.FuncJob(wrappedFunc))
	s.entries[task.Name] = id
}

func (s *CronScheduler) RunNow(name string) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	id, exists := s.entries[name]
	if !exists {
		panic(ErrTaskNotFound)
	}

	s.cron.Entry(id).WrappedJob.Run()
}

// Start starts the cron scheduler.
func (s *CronScheduler) Start() error {
	s.cron.Start()
	return nil
}

// Stop stops the cron scheduler and waits for running tasks to complete.
func (s *CronScheduler) Stop(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.cron.Stop().Done():
		return nil
	}
}

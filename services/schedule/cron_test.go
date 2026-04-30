package schedule

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronScheduler(t *testing.T) {
	s := NewCronScheduler()
	assert.Equal(t, "schedule", s.Name())

	var executed atomic.Int32
	task := Task{
		Name:      "test-task",
		Scheduled: NewIntervalSchedule(time.Millisecond*10, time.Second),
		Run: func(ctx context.Context) error {
			executed.Add(1)
			return nil
		},
	}

	s.Schedule(task)

	// RunNow
	s.RunNow("test-task")
	assert.Equal(t, int32(1), executed.Load())

	// RunNow not found
	assert.PanicsWithValue(t, ErrTaskNotFound, func() {
		s.RunNow("no-such-task")
	})

	// Duplicate task panic
	assert.PanicsWithValue(t, ErrTaskAdded, func() {
		s.Schedule(task)
	})

	// Start and execution
	require.NoError(t, s.Start())
	defer s.Stop(context.Background())

	assert.Eventually(t, func() bool {
		return executed.Load() > 0
	}, time.Second, time.Millisecond*20)
}

func TestCronScheduler_Stop(t *testing.T) {
	s := NewCronScheduler()
	require.NoError(t, s.Start())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := s.Stop(ctx)
	assert.NoError(t, err)
}

func TestCronScheduler_TaskError(t *testing.T) {
	s := NewCronScheduler()
	
	var executed atomic.Int32
	task := Task{
		Name:      "error-task",
		Scheduled: NewIntervalSchedule(time.Millisecond*10, time.Second),
		Run: func(ctx context.Context) error {
			executed.Add(1)
			return assert.AnError
		},
	}

	s.Schedule(task)
	require.NoError(t, s.Start())
	defer s.Stop(context.Background())

	assert.Eventually(t, func() bool {
		return executed.Load() > 0
	}, time.Second, time.Millisecond*20)
}

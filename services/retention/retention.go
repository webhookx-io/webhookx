package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/webhookx-io/webhookx/config/modules"
	"github.com/webhookx-io/webhookx/db"
	"github.com/webhookx-io/webhookx/services/schedule"
	"github.com/webhookx-io/webhookx/utils"
	"go.uber.org/zap"
)

const (
	Interval  = time.Hour * 24
	BatchSize = 1000

	EventsKey   = "events"
	AttemptsKey = "attempts"
)

type RetentionService struct {
	cfg       modules.RetentionConfig
	db        *db.DB
	log       *zap.SugaredLogger
	scheduler schedule.Scheduler
	ttls      map[string]time.Duration
}

func NewRetentionService(
	cfg modules.RetentionConfig,
	db *db.DB,
	log *zap.SugaredLogger,
	scheduler schedule.Scheduler,
) *RetentionService {

	ttls := make(map[string]time.Duration)
	ttls[EventsKey] = time.Hour * 24 * time.Duration(cfg.Events)
	ttls[AttemptsKey] = time.Hour * 24 * time.Duration(cfg.Attempts)

	return &RetentionService{
		cfg:       cfg,
		db:        db,
		log:       log.Named("retention"),
		scheduler: scheduler,
		ttls:      ttls,
	}
}

func (s *RetentionService) Name() string {
	return "retention"
}

func (s *RetentionService) Start() error {
	if !s.cfg.Enabled {
		return nil
	}

	s.log.Infow("service started",
		"purge_interval", utils.FormatDuration(Interval),
		"batch_size", BatchSize,
		zap.Any("ttl", map[string]string{
			"events":   fmt.Sprintf("%dd", s.cfg.Events),
			"attempts": fmt.Sprintf("%dd", s.cfg.Attempts),
		}))

	s.scheduler.Schedule(schedule.Task{
		Name:      "retention",
		Scheduled: schedule.NewIntervalSchedule(0, Interval),
		Run: func(ctx context.Context) error {
			return s.run(ctx)
		},
	})

	return nil
}

func (s *RetentionService) Stop(ctx context.Context) error {
	return nil // retention is managed by scheduler
}

func (s *RetentionService) run(ctx context.Context) error {
	if s.ttls[EventsKey] > 0 {
		ttl := s.ttls[EventsKey]
		count, err := s.purgeEvents(ctx, ttl)
		if err != nil {
			return err
		}
		if count > 0 {
			s.log.Infof("deleted %d expired events", count)
		}
	}

	if s.ttls[AttemptsKey] > 0 {
		ttl := s.ttls[AttemptsKey]
		count, err := s.purgeAttempts(ctx, ttl)
		if err != nil {
			return err
		}
		if count > 0 {
			s.log.Infof("deleted %d expired attempts", count)
		}
	}

	return nil
}

func (s *RetentionService) purgeEvents(ctx context.Context, ttl time.Duration) (int64, error) {
	batchSize := BatchSize

	s.log.Debugw("deleting expired events", "ttl", utils.FormatDuration(ttl), "batch_size", batchSize)

	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		deleted, err := s.db.Events.DeleteTTL(ctx, ttl, batchSize)
		if err != nil {
			return total, fmt.Errorf("failed to delete expired events: %w", err)
		}

		total += deleted

		if deleted < int64(batchSize) {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	return total, nil
}

func (s *RetentionService) purgeAttempts(ctx context.Context, ttl time.Duration) (int64, error) {
	batchSize := BatchSize

	s.log.Debugw("deleting expired attempts", "ttl", utils.FormatDuration(ttl), "batch_size", batchSize)

	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		deleted, err := s.db.Attempts.DeleteTTL(ctx, ttl, batchSize)
		if err != nil {
			return total, fmt.Errorf("failed to delete expired attempts: %w", err)
		}

		total += deleted

		if deleted < int64(batchSize) {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	return total, nil
}

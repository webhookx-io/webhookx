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
	BatchSize = 1000
)

type RetentionService struct {
	cfg       modules.RetentionConfig
	db        *db.DB
	log       *zap.SugaredLogger
	scheduler schedule.Scheduler
}

func NewRetentionService(
	cfg modules.RetentionConfig,
	db *db.DB,
	log *zap.SugaredLogger,
	scheduler schedule.Scheduler,
) *RetentionService {
	return &RetentionService{
		cfg:       cfg,
		db:        db,
		log:       log.Named("retention"),
		scheduler: scheduler,
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
		"interval", utils.FormatDuration(time.Duration(s.cfg.Interval)),
		"batch_size", BatchSize,
		zap.Any("ttl", map[string]string{
			"events":   s.cfg.TTL.Events.String(),
			"attempts": s.cfg.TTL.Attempts.String(),
		}))

	s.scheduler.Schedule(schedule.Task{
		Name:      "retention",
		Scheduled: schedule.NewIntervalSchedule(0, time.Duration(s.cfg.Interval)),
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
	if ttl := time.Duration(s.cfg.TTL.Events); ttl > 0 {
		count, err := s.purgeEvents(ctx, ttl)
		if err != nil {
			return err
		}
		if count > 0 {
			s.log.Infof("deleted %d expired events", count)
		}
	}

	if ttl := time.Duration(s.cfg.TTL.Attempts); ttl > 0 {
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

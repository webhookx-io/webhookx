package circuitbreaker

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/tracing"
	"github.com/webhookx-io/webhookx/worker/circuitbreaker/metrics"
	"go.uber.org/zap"
)

const (
	defaultFlushInterval = time.Second * 10

	timeFormatHour   = "2006-01-02T15Z"
	timeFormatMinute = "2006-01-02T15:04Z"
)

var (
	DefaultFlushInterval = defaultFlushInterval
	cacheKey             = constants.CacheKey{Name: "cb", Version: "v1"}
)

type Option func(m *Manager)

func WithTimeWindowSize(seconds int) Option {
	return func(m *Manager) { m.timeWindow = time.Duration(seconds) * time.Second }
}

func WithFailureRateThreshold(failureRateThreshold int) Option {
	return func(m *Manager) { m.failureRateThreshold = failureRateThreshold }
}

func WithMinimumRequestThreshold(minimumRequestThreshold int) Option {
	return func(m *Manager) { m.minimumRequestThreshold = minimumRequestThreshold }
}

func WithRedisClient(client *redis.Client) Option {
	return func(m *Manager) { m.client = client }
}

func WithFlushInterval(flushInterval time.Duration) Option {
	return func(m *Manager) { m.flushInterval = flushInterval }
}

func WithNowFunc(now func() time.Time) Option {
	return func(m *Manager) { m.now = now }
}

func WithEnabled(enabled bool) Option {
	return func(m *Manager) { m.enabled = enabled }
}

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	log    *zap.SugaredLogger

	enabled                 bool
	client                  *redis.Client
	timeWindow              time.Duration
	failureRateThreshold    int
	minimumRequestThreshold int
	flushInterval           time.Duration
	now                     func() time.Time

	flushMux sync.Mutex

	cache map[string]*Recorder
	mux   sync.RWMutex
}

func NewManager(opts ...Option) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		ctx:                     ctx,
		cancel:                  cancel,
		done:                    make(chan struct{}),
		log:                     zap.S().Named("circuitbreaker"),
		cache:                   make(map[string]*Recorder),
		flushInterval:           DefaultFlushInterval,
		timeWindow:              3600,
		failureRateThreshold:    80,
		minimumRequestThreshold: 100,
		enabled:                 true,
		now:                     time.Now,
	}

	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

func (m *Manager) Record(time time.Time, id string, event metrics.Event) {
	if !m.enabled {
		return
	}

	r := m.getOrCreate(id)
	r.Record(time.Unix(), event)
}

func (m *Manager) getOrCreate(id string) *Recorder {
	m.mux.RLock()
	instance := m.cache[id]
	m.mux.RUnlock()
	if instance != nil {
		return instance
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	if instance = m.cache[id]; instance == nil {
		instance = NewRecorder(60)
		m.cache[id] = instance
	}
	return instance
}

func (m *Manager) Start() {
	defer close(m.done)

	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if err := m.Flush(context.TODO()); err != nil {
				m.log.Errorf("failed to call flush: %s", err)
			}
		}
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.cancel()
	<-m.done
	if err := m.Flush(ctx); err != nil {
		return fmt.Errorf("failed to stop: %s", err)
	}
	return nil
}

func (m *Manager) Flush(ctx context.Context) error {
	if !m.enabled {
		return nil
	}

	m.flushMux.Lock()
	defer m.flushMux.Unlock()

	ctx, span := tracing.Start(ctx, "circuitbreaker.flush")
	defer span.End()

	m.mux.RLock()

	flushTimeEnd := m.now().Unix() - 1
	data := make(map[string]TimeBucketMetric)
	recorders := make([]*Recorder, 0, len(m.cache))

	for id, r := range m.cache {
		flushTimeStart := r.LastSync() + 1
		// aggregate by hour
		hourMetrics := r.Aggregate(Hour, flushTimeStart, flushTimeEnd)
		for _, m := range hourMetrics {
			key := cacheKey.Build(id, ":", time.Unix(m.Start, 0).UTC().Format(timeFormatHour))
			data[key] = m
		}
		// aggregate by minute
		minuteMetrics := r.Aggregate(Minute, flushTimeStart, flushTimeEnd)
		for _, m := range minuteMetrics {
			key := cacheKey.Build(id, ":", time.Unix(m.Start, 0).UTC().Format(timeFormatMinute))
			data[key] = m
		}

		recorders = append(recorders, r)
	}
	m.mux.RUnlock()

	// verbose
	// m.log.Debugw("flushing memory data to redis", "metrics", data)

	if len(data) > 0 {
		pipeline := m.client.Pipeline()
		for key, s := range data {
			if s.Success > 0 {
				pipeline.HIncrBy(ctx, key, "success", s.Success)
			}
			if s.Error > 0 {
				pipeline.HIncrBy(ctx, key, "failure", s.Error)
			}
			ttl := time.Minute * 61
			if s.Until-s.Start >= 3600 {
				ttl = time.Hour * 25
			}
			pipeline.Expire(ctx, key, ttl)
		}
		_, err := pipeline.Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to flush memory data to redis: %w", err)
		}
	}

	for _, r := range recorders {
		r.SetLastSync(flushTimeEnd)
	}

	return nil
}

func (m *Manager) GetCircuitBreaker(ctx context.Context, id string) (CircuitBreaker, error) {
	cb := &circuitBreaker{
		name:  id,
		state: StateClosed,
	}

	now := m.now()

	pipeline := m.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, 0)
	metrics := make([]TimeBucketMetric, 0)

	if m.timeWindow < time.Hour {
		start := now.Add(-m.timeWindow).Truncate(time.Minute)
		end := now.Truncate(time.Minute)
		for t := start; !t.After(end); t = t.Add(time.Minute) {
			minuteKey := cacheKey.Build(id, ":", t.UTC().Format(timeFormatMinute))
			cmds = append(cmds, pipeline.HGetAll(ctx, minuteKey))
			metrics = append(metrics, TimeBucketMetric{
				Start: t.Unix(),
				Until: min(t.Add(time.Minute).Unix(), now.Unix()),
			})
		}
	} else {
		start := now.Add(-m.timeWindow).Truncate(time.Hour)
		end := now.Truncate(time.Hour)
		for t := start; !t.After(end); t = t.Add(time.Hour) {
			hourKey := cacheKey.Build(id, ":", t.UTC().Format(timeFormatHour))
			cmds = append(cmds, pipeline.HGetAll(ctx, hourKey))
			metrics = append(metrics, TimeBucketMetric{
				Start: t.Unix(),
				Until: min(t.Add(time.Hour).Unix(), now.Unix()),
			})
		}
	}

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, err
	}

	for i, cmd := range cmds {
		v, err := cmd.Result()
		if err != nil {
			return nil, err
		}

		success, _ := strconv.Atoi(v["success"])
		failure, _ := strconv.Atoi(v["failure"])

		m := &metrics[i]
		m.Success += int64(success)
		m.Error += int64(failure)
	}

	metric := timeProrate(metrics, now.Unix(), int64(m.timeWindow.Seconds()))
	cb.metric = metric

	failureRate := float64(m.failureRateThreshold) / 100.0
	if metric.TotalRequest() >= int64(m.minimumRequestThreshold) &&
		metric.FailureRate() >= failureRate {
		cb.state = StateOpen
	}
	return cb, nil
}

func timeProrate(metrics []TimeBucketMetric, now int64, windowSize int64) TimeBucketMetric {
	windowStart := now - windowSize
	var success float64
	var error float64

	for _, m := range metrics {
		overlapStart := max(m.Start, windowStart)
		overlapEnd := min(m.Until, now)

		overlap := overlapEnd - overlapStart
		if overlap <= 0 {
			continue
		}

		bucketSize := m.Until - m.Start
		weight := float64(overlap) / float64(bucketSize)

		success += float64(m.Success) * weight
		error += float64(m.Error) * weight
	}

	return TimeBucketMetric{
		Start:   windowStart,
		Until:   now,
		Success: int64(success),
		Error:   int64(error),
	}
}

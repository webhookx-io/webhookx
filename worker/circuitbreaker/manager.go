package circuitbreaker

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/webhookx-io/webhookx/constants"
	"github.com/webhookx-io/webhookx/pkg/tracing"
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

type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	log    *zap.SugaredLogger

	client                  *redis.Client
	timeWindow              time.Duration
	failureRateThreshold    int
	minimumRequestThreshold int
	flushInterval           time.Duration

	mux   sync.RWMutex
	cache map[string]*Recorder
}

func NewManager(opts ...Option) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		ctx:           ctx,
		cancel:        cancel,
		log:           zap.S().Named("circuitbreaker"),
		cache:         make(map[string]*Recorder),
		flushInterval: DefaultFlushInterval,
	}

	for _, opt := range opts {
		opt(manager)
	}

	return manager
}

func (m *Manager) Record(id string, outcome Outcome) {
	ts := time.Now().Unix()
	r := m.getOrCreate(id)
	r.Record(ts, outcome)
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
		instance = NewRecorder()
		instance.SetLastSync(time.Now().Unix() - 1)
		m.cache[id] = instance
	}
	return instance
}

func (m *Manager) Start() {
	ticker := time.NewTicker(m.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.flush(context.TODO())
		}
	}
}

func (m *Manager) Stop() {
	m.cancel()
	m.flush(context.TODO())
}

func (m *Manager) flush(ctx context.Context) {
	ctx, span := tracing.Start(ctx, "circuitbreaker.flush")
	defer span.End()

	m.mux.RLock()

	flushTimeEnd := time.Now().Unix() - 1
	data := make(map[string]Stats)
	recorders := make([]*Recorder, 0, len(m.cache))

	for id, r := range m.cache {
		flushTimeStart := r.LastSync() + 1
		// aggr by hour
		stats := r.Aggregate(time.Hour, flushTimeStart, flushTimeEnd)
		for _, s := range stats {
			key := cacheKey.Build(id, ":", time.Unix(s.StartTime, 0).UTC().Format(timeFormatHour))
			data[key] = s
		}
		// aggr by minute
		stats = r.Aggregate(time.Minute, flushTimeStart, flushTimeEnd)
		for _, s := range stats {
			key := cacheKey.Build(id, ":", time.Unix(s.StartTime, 0).UTC().Format(timeFormatMinute))
			data[key] = s
		}

		recorders = append(recorders, r)
	}
	m.mux.RUnlock()

	m.log.Debugw("flushing memory data to redis", "stats", data)

	if len(data) > 0 {
		pipeline := m.client.Pipeline()
		for key, s := range data {
			if s.TotalSuccess() > 0 {
				pipeline.HIncrBy(ctx, key, "success", s.TotalSuccess())
			}
			if s.TotalFailures() > 0 {
				pipeline.HIncrBy(ctx, key, "failure", s.TotalFailures())
			}
			ttl := time.Hour + time.Minute
			if s.EndTime-s.StartTime >= 3600 { // hour stats
				ttl = time.Hour*24 + time.Minute
			}
			pipeline.Expire(ctx, key, ttl)
		}
		_, err := pipeline.Exec(ctx)
		if err != nil {
			m.log.Warnf("failed to flush memory data to redis: %s", err.Error())
			return
		}
	}

	for _, r := range recorders {
		r.SetLastSync(flushTimeEnd)
	}
}

func (m *Manager) Get(ctx context.Context, id string) (CircuitBreaker, error) {
	cb := &CircuitBreakerImpl{
		state: StateClosed,
	}

	now := time.Now()

	pipeline := m.client.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, 0)
	stats := make([]Stats, 0)

	if m.timeWindow < time.Hour {
		start := now.Add(-m.timeWindow).Truncate(time.Minute)
		end := now.Truncate(time.Minute)
		for t := start; !t.After(end); t = t.Add(time.Minute) {
			minuteKey := cacheKey.Build(id, ":", t.UTC().Format(timeFormatMinute))
			cmds = append(cmds, pipeline.HGetAll(ctx, minuteKey))
			stats = append(stats, Stats{
				StartTime: t.Unix(),
				EndTime:   min(t.Add(time.Minute).Unix(), now.Unix()),
			})
		}
	} else {
		start := now.Add(-m.timeWindow).Truncate(time.Hour)
		end := now.Truncate(time.Hour)
		for t := start; !t.After(end); t = t.Add(time.Hour) {
			hourKey := cacheKey.Build(id, ":", t.UTC().Format(timeFormatHour))
			cmds = append(cmds, pipeline.HGetAll(ctx, hourKey))
			stats = append(stats, Stats{
				StartTime: t.Unix(),
				EndTime:   min(t.Add(time.Hour).Unix(), now.Unix()),
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

		s := &stats[i]
		s.Success = int64(success)
		s.Failure = int64(failure)
	}

	s := timeWeightedCalc(stats, now.Unix(), int64(m.timeWindow.Seconds()))
	cb.stats = s

	failureRate := float64(m.failureRateThreshold) / 100.0
	if s.TotalRequest() > int64(m.minimumRequestThreshold) &&
		s.FailureRate() >= failureRate {
		cb.state = StateOpen
	}
	return cb, nil
}

func timeWeightedCalc(stats []Stats, now int64, windowSize int64) Stats {
	windowStart := now - windowSize
	var success float64
	var failure float64

	for _, s := range stats {
		overlapStart := max(s.StartTime, windowStart)
		overlapEnd := min(s.EndTime, now)

		overlap := overlapEnd - overlapStart
		if overlap <= 0 {
			continue
		}

		bucketSize := s.EndTime - s.StartTime
		weight := float64(overlap) / float64(bucketSize)

		success += float64(s.Success) * weight
		failure += float64(s.Failure) * weight
	}

	return Stats{
		StartTime: windowStart,
		EndTime:   now,
		Success:   int64(success),
		Failure:   int64(failure),
	}
}

package loglimiter

import (
	"sync"
	"time"
)

type Limiter struct {
	mux    sync.Mutex
	window time.Duration
	logs   map[string]time.Time
}

func NewLimiter(window time.Duration) *Limiter {
	return &Limiter{
		window: window,
		logs:   make(map[string]time.Time),
	}
}

func (l *Limiter) Allow(key string) bool {
	l.mux.Lock()
	defer l.mux.Unlock()

	now := time.Now()
	last := l.logs[key]
	if now.Sub(last) > l.window {
		l.logs[key] = now
		return true
	}

	return false
}

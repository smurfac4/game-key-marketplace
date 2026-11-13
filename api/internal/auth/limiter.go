package auth

import (
	"sync"
	"time"
)

type bucket struct {
	Count int
	Until time.Time
}
type Limiter struct {
	mu          sync.Mutex
	entries     map[string]bucket
	now         func() time.Time
	nextCleanup time.Time
	limit       int
	window      time.Duration
}

func NewLimiter() *Limiter { return NewWindowLimiter(10, 15*time.Minute) }

func NewWindowLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{entries: map[string]bucket{}, now: time.Now, limit: limit, window: window}
}

func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if !now.Before(l.nextCleanup) {
		for k, v := range l.entries {
			if !now.Before(v.Until) {
				delete(l.entries, k)
			}
		}
		l.nextCleanup = now.Add(time.Minute)
	}
	b, ok := l.entries[key]
	if !ok || !now.Before(b.Until) {
		if !ok && len(l.entries) >= 10000 {
			return false, time.Minute
		}
		b = bucket{Until: now.Add(l.window)}
	}
	if b.Count >= l.limit {
		return false, b.Until.Sub(now)
	}
	b.Count++
	l.entries[key] = b
	return true, 0
}

package api

import (
	"sync"
	"time"
)

type IPRateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	clients map[string]rateWindow
}

type rateWindow struct {
	count   int
	resetAt time.Time
}

func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &IPRateLimiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		clients: make(map[string]rateWindow),
	}
}

func (l *IPRateLimiter) Allow(key string) (bool, time.Duration) {
	if key == "" {
		key = "unknown"
	}
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.clients[key]
	if !ok || now.After(entry.resetAt) {
		l.clients[key] = rateWindow{
			count:   1,
			resetAt: now.Add(l.window),
		}
		l.pruneLocked(now)
		return true, 0
	}

	if entry.count >= l.limit {
		retryAfter := time.Until(entry.resetAt)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	entry.count++
	l.clients[key] = entry
	return true, 0
}

func (l *IPRateLimiter) pruneLocked(now time.Time) {
	// Keep memory bounded by dropping expired windows opportunistically.
	if len(l.clients) < 4096 {
		return
	}
	for key, entry := range l.clients {
		if now.After(entry.resetAt) {
			delete(l.clients, key)
		}
	}
}

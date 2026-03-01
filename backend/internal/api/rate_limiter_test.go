package api

import (
	"testing"
	"time"
)

func TestIPRateLimiterAllow(t *testing.T) {
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	now := base

	limiter := NewIPRateLimiter(2, 10*time.Second)
	limiter.now = func() time.Time { return now }

	if ok, _ := limiter.Allow("1.2.3.4"); !ok {
		t.Fatal("first request should be allowed")
	}
	if ok, _ := limiter.Allow("1.2.3.4"); !ok {
		t.Fatal("second request should be allowed")
	}
	ok, retryAfter := limiter.Allow("1.2.3.4")
	if ok {
		t.Fatal("third request should be rate limited")
	}
	if retryAfter <= 0 {
		t.Fatal("retry-after should be positive")
	}

	now = base.Add(11 * time.Second)
	if ok, _ := limiter.Allow("1.2.3.4"); !ok {
		t.Fatal("request after window should be allowed")
	}
}

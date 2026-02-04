package crawler

import (
	"testing"
	"time"
)

func TestHostStateCircuitBreaker(t *testing.T) {
	hs := NewHostState("example.com", 2, 2, 50*time.Millisecond)
	if !hs.Allow() {
		t.Fatal("expected allow in closed state")
	}
	hs.OnResult(false)
	hs.OnResult(false)
	if hs.State() != CircuitOpen {
		t.Fatalf("expected circuit open, got %s", hs.State())
	}
	if hs.Allow() {
		t.Fatal("expected disallow while open")
	}
	time.Sleep(60 * time.Millisecond)
	if !hs.Allow() {
		t.Fatal("expected allow after reset")
	}
}
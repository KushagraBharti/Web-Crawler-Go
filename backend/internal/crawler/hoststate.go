package crawler

import (
	"sync"
	"time"
)

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half_open"
)

type HostState struct {
	Host        string
	Semaphore   *Semaphore
	Circuit     CircuitState
	ErrCount    int
	LastFail    time.Time
	OpenedAt    time.Time
	TripCount   int
	ResetAfter  time.Duration
	mu          sync.Mutex
}

func NewHostState(host string, perHost int, tripCount int, reset time.Duration) *HostState {
	return &HostState{
		Host:       host,
		Semaphore:  NewSemaphore(perHost),
		Circuit:    CircuitClosed,
		TripCount:  tripCount,
		ResetAfter: reset,
	}
}

func (h *HostState) Allow() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch h.Circuit {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(h.OpenedAt) > h.ResetAfter {
			h.Circuit = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return true
	}
}

func (h *HostState) OnResult(success bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if success {
		h.ErrCount = 0
		if h.Circuit != CircuitClosed {
			h.Circuit = CircuitClosed
		}
		return
	}
	h.ErrCount++
	h.LastFail = time.Now()
	if h.ErrCount >= h.TripCount {
		h.Circuit = CircuitOpen
		h.OpenedAt = time.Now()
	}
}

func (h *HostState) State() CircuitState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Circuit
}
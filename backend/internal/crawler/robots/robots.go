package robots

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

type State string

const (
	StateUnknown  State = "unknown"
	StateFetching State = "fetching"
	StateReady    State = "ready"
	StateError    State = "error"
)

type Manager struct {
	mu        sync.Mutex
	entries   map[string]*entry
	client    *http.Client
	userAgent string
	ttl       time.Duration
	fetchSem  chan struct{}
}

type entry struct {
	group    *robotstxt.Group
	expires  time.Time
	ready    bool
	fetching bool
	state    State
	readyCh  chan struct{}
}

func New(client *http.Client, userAgent string, ttl time.Duration, maxConcurrent int) *Manager {
	if maxConcurrent <= 0 {
		maxConcurrent = 4
	}
	return &Manager{
		entries:   make(map[string]*entry),
		client:    client,
		userAgent: userAgent,
		ttl:       ttl,
		fetchSem:  make(chan struct{}, maxConcurrent),
	}
}

func (m *Manager) Allowed(ctx context.Context, target *url.URL) (allowed bool, ready bool, state State, err error) {
	host := hostKey(target)
	now := time.Now()

	m.mu.Lock()
	e := m.entries[host]
	if e != nil && e.ready && now.Before(e.expires) {
		group := e.group
		state = e.state
		m.mu.Unlock()
		if group == nil {
			return true, true, state, nil
		}
		return group.Test(target.Path), true, state, nil
	}
	if e != nil && e.fetching {
		state = e.state
		m.mu.Unlock()
		return false, false, state, nil
	}
	if e == nil {
		e = &entry{state: StateFetching}
		m.entries[host] = e
	}
	e.fetching = true
	e.ready = false
	e.state = StateFetching
	e.readyCh = make(chan struct{})
	m.mu.Unlock()

	go m.fetch(ctx, host, target.Scheme)
	return false, false, StateFetching, nil
}

func (m *Manager) State(host string) State {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.entries[host]; ok {
		return e.state
	}
	return StateUnknown
}

func (m *Manager) fetch(ctx context.Context, host, scheme string) {
	m.fetchSem <- struct{}{}
	defer func() { <-m.fetchSem }()

	robotsURL := scheme + "://" + host + "/robots.txt"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		m.finish(host, nil, StateError)
		return
	}
	req.Header.Set("User-Agent", m.userAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		m.finish(host, nil, StateError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		io.Copy(io.Discard, resp.Body)
		m.finish(host, nil, StateReady)
		return
	}

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		m.finish(host, nil, StateError)
		return
	}
	group := data.FindGroup(m.userAgent)
	m.finish(host, group, StateReady)
}

func (m *Manager) finish(host string, group *robotstxt.Group, state State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[host]
	if e == nil {
		return
	}
	e.group = group
	e.ready = true
	e.fetching = false
	e.state = state
	e.expires = time.Now().Add(m.ttl)
	if e.readyCh != nil {
		close(e.readyCh)
	}
}

func hostKey(u *url.URL) string {
	h := strings.ToLower(u.Host)
	h = strings.TrimSuffix(h, ".")
	return h
}
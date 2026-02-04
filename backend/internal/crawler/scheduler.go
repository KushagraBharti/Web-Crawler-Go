package crawler

import (
	"context"
	"net/url"
	"sync"
	"time"

	"webcrawler/internal/crawler/robots"
)

type Scheduler struct {
	ctx           context.Context
	in            chan *Task
	out           chan *Task
	frontierLimit int
	globalSem     *Semaphore
	perHost       int
	tripCount     int
	circuitReset  time.Duration
	respectRobots bool
	robots        *robots.Manager

	hostQueues map[string][]*Task
	hosts      []string
	hostIndex  int
	hostStates map[string]*HostState
	frontierSz int
	mu         sync.RWMutex
}

func NewScheduler(ctx context.Context, in chan *Task, out chan *Task, frontierLimit int, global *Semaphore, perHost int, tripCount int, circuitReset time.Duration, respectRobots bool, robotsMgr *robots.Manager) *Scheduler {
	return &Scheduler{
		ctx:           ctx,
		in:            in,
		out:           out,
		frontierLimit: frontierLimit,
		globalSem:     global,
		perHost:       perHost,
		tripCount:     tripCount,
		circuitReset:  circuitReset,
		respectRobots: respectRobots,
		robots:        robotsMgr,
		hostQueues:    make(map[string][]*Task),
		hostStates:    make(map[string]*HostState),
	}
}

func (s *Scheduler) Run() {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case task := <-s.in:
			s.enqueue(task)
		case <-ticker.C:
			s.schedule()
		}
	}
}

func (s *Scheduler) enqueue(task *Task) {
	if task == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.frontierLimit > 0 && s.frontierSz >= s.frontierLimit {
		return
	}
	s.frontierSz++
	queue := s.hostQueues[task.Host]
	if len(queue) == 0 {
		s.hosts = append(s.hosts, task.Host)
	}
	s.hostQueues[task.Host] = append(queue, task)
	if _, ok := s.hostStates[task.Host]; !ok {
		s.hostStates[task.Host] = NewHostState(task.Host, s.perHost, s.tripCount, s.circuitReset)
	}
}

func (s *Scheduler) schedule() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.hosts) == 0 {
		return
	}

	iterations := len(s.hosts)
	for i := 0; i < iterations; i++ {
		if len(s.hosts) == 0 {
			return
		}
		if s.hostIndex >= len(s.hosts) {
			s.hostIndex = 0
		}
		host := s.hosts[s.hostIndex]
		queue := s.hostQueues[host]
		if len(queue) == 0 {
			s.removeHostAt(s.hostIndex)
			continue
		}
		task := queue[0]
		if !task.NotBefore.IsZero() && time.Now().Before(task.NotBefore) {
			s.hostIndex++
			continue
		}
		state := s.hostStates[host]
		if state != nil && !state.Allow() {
			task.NotBefore = time.Now().Add(500 * time.Millisecond)
			s.hostQueues[host][0] = task
			s.hostIndex++
			continue
		}
		if !s.globalSem.TryAcquire() {
			return
		}
		if state != nil && !state.Semaphore.TryAcquire() {
			s.globalSem.Release()
			s.hostIndex++
			continue
		}
		if s.respectRobots && s.robots != nil {
			parsed, err := url.Parse(task.URL)
			if err != nil {
				state.Semaphore.Release()
				s.globalSem.Release()
				s.hostQueues[host] = s.hostQueues[host][1:]
				s.frontierSz--
				s.hostIndex++
				continue
			}
			allowed, ready, _, _ := s.robots.Allowed(s.ctx, parsed)
			if !ready {
				state.Semaphore.Release()
				s.globalSem.Release()
				task.NotBefore = time.Now().Add(750 * time.Millisecond)
				s.hostQueues[host][0] = task
				s.hostIndex++
				continue
			}
			if !allowed {
				state.Semaphore.Release()
				s.globalSem.Release()
				s.hostQueues[host] = s.hostQueues[host][1:]
				s.frontierSz--
				s.hostIndex++
				continue
			}
		}
		// dequeue
		s.hostQueues[host] = s.hostQueues[host][1:]
		s.frontierSz--
		task.Permit = &Permit{Global: s.globalSem, Host: state.Semaphore}
		select {
		case s.out <- task:
			// ok
		default:
			// backpressure, requeue
			task.Permit.Release()
			task.NotBefore = time.Now().Add(200 * time.Millisecond)
			s.hostQueues[host] = append([]*Task{task}, s.hostQueues[host]...)
			s.frontierSz++
		}
		s.hostIndex++
	}
}

func (s *Scheduler) removeHostAt(idx int) {
	host := s.hosts[idx]
	delete(s.hostQueues, host)
	s.hosts = append(s.hosts[:idx], s.hosts[idx+1:]...)
	if s.hostIndex >= len(s.hosts) {
		s.hostIndex = 0
	}
}

func (s *Scheduler) FrontierSize() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.frontierSz
}

func (s *Scheduler) HostState(host string) *HostState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hostStates[host]
}

func (s *Scheduler) HostStatesSnapshot() map[string]*HostState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*HostState, len(s.hostStates))
	for k, v := range s.hostStates {
		out[k] = v
	}
	return out
}
 

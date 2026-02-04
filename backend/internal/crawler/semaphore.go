package crawler

type Semaphore struct {
	ch chan struct{}
}

func NewSemaphore(size int) *Semaphore {
	if size <= 0 {
		size = 1
	}
	return &Semaphore{ch: make(chan struct{}, size)}
}

func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Semaphore) Release() {
	select {
	case <-s.ch:
	default:
	}
}

func (s *Semaphore) Inflight() int {
	return len(s.ch)
}

func (s *Semaphore) Capacity() int {
	return cap(s.ch)
}

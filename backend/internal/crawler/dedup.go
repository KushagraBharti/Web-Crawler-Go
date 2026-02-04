package crawler

import "sync"

type Deduper struct {
	shards []dedupShard
}

type dedupShard struct {
	mu sync.RWMutex
	m  map[string]struct{}
}

func NewDeduper(shards int) *Deduper {
	if shards <= 0 {
		shards = 32
	}
	d := &Deduper{shards: make([]dedupShard, shards)}
	for i := range d.shards {
		d.shards[i].m = make(map[string]struct{})
	}
	return d
}

func (d *Deduper) Seen(canonical string) bool {
	if canonical == "" {
		return true
	}
	idx := fnv32(canonical) % uint32(len(d.shards))
	sh := &d.shards[idx]
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if _, ok := sh.m[canonical]; ok {
		return true
	}
	sh.m[canonical] = struct{}{}
	return false
}

func fnv32(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	hash := uint32(offset32)
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= prime32
	}
	return hash
}
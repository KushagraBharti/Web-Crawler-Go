package metrics

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"webcrawler/internal/crawler/robots"
)

type FetchEvent struct {
	Host       string
	LatencyMS  int64
	Bytes      int64
	ReusedConn bool
	ErrClass   string
}

type EdgeEvent struct {
	Src string
	Dst string
}

type Frame struct {
	Ts         time.Time   `json:"ts"`
	Throughput Throughput  `json:"throughput"`
	Queues     QueueDepths `json:"queues"`
	Errors     []ErrCount  `json:"errors"`
	Hosts      []HostFrame `json:"hosts"`
	GraphDelta GraphDelta  `json:"graph_delta"`
}

type Throughput struct {
	PagesPerSec float64 `json:"pages_per_sec"`
}

type QueueDepths struct {
	Frontier int `json:"frontier"`
	Fetch    int `json:"fetch"`
	Parse    int `json:"parse"`
}

type ErrCount struct {
	Class string `json:"class"`
	Count int    `json:"count"`
}

type HostFrame struct {
	Host        string `json:"host"`
	Inflight    int    `json:"inflight"`
	P95Ms       int    `json:"p95_ms"`
	ErrorRate   float64 `json:"error_rate"`
	ReuseRate   float64 `json:"reuse_rate"`
	RobotsState string `json:"robots_state"`
	Circuit     string `json:"circuit_state"`
}

type HostSnapshot struct {
	Inflight int
	Circuit  string
}

type GraphDelta struct {
	Nodes []string   `json:"nodes"`
	Edges [][3]any   `json:"edges"`
}

type Telemetry struct {
	fetchCh     chan FetchEvent
	edgesCh     chan EdgeEvent
	queueGetter func() (int, int, int)
	hostGetter  func() map[string]HostSnapshot
	robots      *robots.Manager

	mu          sync.Mutex
	subscribers map[int]chan Frame
	nextID      int

	hostStats   map[string]*hostMetrics
	errorCounts map[string]int
	nodesSeen   map[string]struct{}
	edgesSeen   map[string]int

	intervalPages int
}

type hostMetrics struct {
	latencies []int
	reqs      int
	errs      int
	reuse     int
}

func NewTelemetry() *Telemetry {
	return &Telemetry{
		fetchCh:     make(chan FetchEvent, 2048),
		edgesCh:     make(chan EdgeEvent, 2048),
		subscribers: make(map[int]chan Frame),
		hostStats:   make(map[string]*hostMetrics),
		errorCounts: make(map[string]int),
		nodesSeen:   make(map[string]struct{}),
		edgesSeen:   make(map[string]int),
	}
}

func (t *Telemetry) SetQueueGetter(getter func() (int, int, int)) {
	t.queueGetter = getter
}

func (t *Telemetry) SetHostGetter(getter func() map[string]HostSnapshot) {
	t.hostGetter = getter
}

func (t *Telemetry) SetRobotsManager(mgr *robots.Manager) {
	t.robots = mgr
}

func (t *Telemetry) FetchEvents() chan<- FetchEvent {
	return t.fetchCh
}

func (t *Telemetry) EdgeEvents() chan<- EdgeEvent {
	return t.edgesCh
}

func (t *Telemetry) Subscribe() (<-chan Frame, func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	id := t.nextID
	t.nextID++
	ch := make(chan Frame, 8)
	t.subscribers[id] = ch
	return ch, func() {
		t.mu.Lock()
		if c, ok := t.subscribers[id]; ok {
			delete(t.subscribers, id)
			close(c)
		}
		t.mu.Unlock()
	}
}

func (t *Telemetry) Run(ctx context.Context) {
	frameInterval := 200 * time.Millisecond
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-t.fetchCh:
			t.onFetch(ev)
		case ev := <-t.edgesCh:
			t.onEdge(ev)
		case <-ticker.C:
			t.emitFrame(frameInterval)
		}
	}
}

func (t *Telemetry) onFetch(ev FetchEvent) {
	stats := t.hostStats[ev.Host]
	if stats == nil {
		stats = &hostMetrics{}
		t.hostStats[ev.Host] = stats
	}
	stats.reqs++
	if ev.ErrClass != "" {
		stats.errs++
		t.errorCounts[ev.ErrClass]++
	}
	if ev.ReusedConn {
		stats.reuse++
	}
	if ev.LatencyMS > 0 {
		stats.latencies = append(stats.latencies, int(ev.LatencyMS))
		if len(stats.latencies) > 200 {
			stats.latencies = stats.latencies[len(stats.latencies)-200:]
		}
	}
	if ev.ErrClass == "" {
		t.intervalPages++
	}
}

func (t *Telemetry) onEdge(ev EdgeEvent) {
	if ev.Src == "" || ev.Dst == "" {
		return
	}
	if _, ok := t.nodesSeen[ev.Src]; !ok {
		t.nodesSeen[ev.Src] = struct{}{}
	}
	if _, ok := t.nodesSeen[ev.Dst]; !ok {
		t.nodesSeen[ev.Dst] = struct{}{}
	}
	key := ev.Src + "->" + ev.Dst
	t.edgesSeen[key]++
}

func (t *Telemetry) emitFrame(interval time.Duration) {
	var queues QueueDepths
	if t.queueGetter != nil {
		queues.Frontier, queues.Fetch, queues.Parse = t.queueGetter()
	}
	QueueDepth.WithLabelValues("frontier").Set(float64(queues.Frontier))
	QueueDepth.WithLabelValues("fetch").Set(float64(queues.Fetch))
	QueueDepth.WithLabelValues("parse").Set(float64(queues.Parse))

	hostSnapshot := map[string]HostSnapshot{}
	if t.hostGetter != nil {
		hostSnapshot = t.hostGetter()
	}

	hosts := make([]HostFrame, 0, len(t.hostStats))
	for host, stats := range t.hostStats {
		p95 := 0
		if len(stats.latencies) > 0 {
			lat := append([]int(nil), stats.latencies...)
			sort.Ints(lat)
			idx := int(float64(len(lat)-1) * 0.95)
			p95 = lat[idx]
		}
		errRate := 0.0
		if stats.reqs > 0 {
			errRate = float64(stats.errs) / float64(stats.reqs)
		}
		reuseRate := 0.0
		if stats.reqs > 0 {
			reuseRate = float64(stats.reuse) / float64(stats.reqs)
		}

		frame := HostFrame{Host: host, P95Ms: p95, ErrorRate: errRate, ReuseRate: reuseRate}
		if hs, ok := hostSnapshot[host]; ok {
			frame.Inflight = hs.Inflight
			frame.Circuit = hs.Circuit
		}
		if t.robots != nil {
			frame.RobotsState = string(t.robots.State(host))
		}
		hosts = append(hosts, frame)
	}
	// Sort by inflight descending for UI stability
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Inflight > hosts[j].Inflight })
	if len(hosts) > 25 {
		hosts = hosts[:25]
	}

	errors := make([]ErrCount, 0, len(t.errorCounts))
	for class, count := range t.errorCounts {
		errors = append(errors, ErrCount{Class: class, Count: count})
	}
	sort.Slice(errors, func(i, j int) bool { return errors[i].Count > errors[j].Count })
	if len(errors) > 10 {
		errors = errors[:10]
	}
	// reset error counts each frame to show fresh trends
	t.errorCounts = make(map[string]int)

	nodes := make([]string, 0, len(t.nodesSeen))
	for n := range t.nodesSeen {
		nodes = append(nodes, n)
	}
	edges := make([][3]any, 0, len(t.edgesSeen))
	for key, count := range t.edgesSeen {
		parts := strings.SplitN(key, "->", 2)
		if len(parts) == 2 {
			edges = append(edges, [3]any{parts[0], parts[1], count})
		}
	}
	// reset deltas
	t.nodesSeen = make(map[string]struct{})
	t.edgesSeen = make(map[string]int)

	pagesPerSec := float64(t.intervalPages) / interval.Seconds()
	t.intervalPages = 0

	frame := Frame{
		Ts: time.Now(),
		Throughput: Throughput{PagesPerSec: pagesPerSec},
		Queues: queues,
		Errors: errors,
		Hosts: hosts,
		GraphDelta: GraphDelta{Nodes: nodes, Edges: edges},
	}

	t.mu.Lock()
	for _, ch := range t.subscribers {
		select {
		case ch <- frame:
		default:
		}
	}
	t.mu.Unlock()
}

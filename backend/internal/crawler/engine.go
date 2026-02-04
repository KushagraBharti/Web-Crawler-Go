package crawler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"golang.org/x/net/html"
	"webcrawler/internal/crawler/robots"
	"webcrawler/internal/metrics"
	"webcrawler/internal/storage"
)

type Engine struct {
	runID     uuid.UUID
	cfg       RunConfig
	store     storage.Store
	telemetry *metrics.Telemetry

	ctx    context.Context
	cancel context.CancelFunc

	deduper   *Deduper
	scheduler *Scheduler
	robotsMgr *robots.Manager
	client    *http.Client

	enqueueCh chan *Task
	fetchCh   chan *Task
	parseCh   chan *FetchResult

	pageWrites chan storage.PageRecord
	errorWrites chan errorRecord
	edgeWrites  chan edgeRecord

	startedAt time.Time
	pagesFetched atomic.Int64
	stopOnce sync.Once
}

type errorRecord struct {
	runID   uuid.UUID
	host    string
	url     string
	class   string
	message string
}

type edgeRecord struct {
	runID uuid.UUID
	src   string
	dst   string
	count int
}

func NewEngine(runID uuid.UUID, cfg RunConfig, store storage.Store, telemetry *metrics.Telemetry) *Engine {
	cfg = cfg.Normalize()
	if cfg.GlobalConcurrency <= 0 {
		cfg.GlobalConcurrency = 32
	}
	if cfg.PerHostConcurrency <= 0 {
		cfg.PerHostConcurrency = 2
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20
	}
	if cfg.RetryMax < 0 {
		cfg.RetryMax = 0
	}

	ctx, cancel := context.WithCancel(context.Background())

	globalSem := NewSemaphore(cfg.GlobalConcurrency)
	frontierCap := cfg.GlobalConcurrency * 200
	fetchCap := cfg.GlobalConcurrency * 4
	parseCap := cfg.GlobalConcurrency * 4

	enqueueCh := make(chan *Task, frontierCap)
	fetchCh := make(chan *Task, fetchCap)
	parseCh := make(chan *FetchResult, parseCap)

	client := &http.Client{
		Transport: buildTransport(cfg),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var robotsMgr *robots.Manager
	if cfg.RespectRobots {
		robotsMgr = robots.New(client, cfg.UserAgent, cfg.RobotsTTL, 4)
	}

	scheduler := NewScheduler(ctx, enqueueCh, fetchCh, frontierCap, globalSem, cfg.PerHostConcurrency, cfg.CircuitTripCount, cfg.CircuitResetTime, cfg.RespectRobots, robotsMgr)

	return &Engine{
		runID:      runID,
		cfg:        cfg,
		store:      store,
		telemetry:  telemetry,
		ctx:        ctx,
		cancel:     cancel,
		deduper:    NewDeduper(64),
		scheduler:  scheduler,
		robotsMgr:  robotsMgr,
		client:     client,
		enqueueCh:  enqueueCh,
		fetchCh:    fetchCh,
		parseCh:    parseCh,
		pageWrites: make(chan storage.PageRecord, 2048),
		errorWrites: make(chan errorRecord, 1024),
		edgeWrites: make(chan edgeRecord, 1024),
	}
}

func buildTransport(cfg RunConfig) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   cfg.HeaderTimeout,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cfg.GlobalConcurrency * 4,
		MaxIdleConnsPerHost:   max(cfg.PerHostConcurrency*2, 8),
		MaxConnsPerHost:       max(cfg.PerHostConcurrency*4, 16),
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.HeaderTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func (e *Engine) Start(seed string) {
	e.startedAt = time.Now()
	if e.telemetry != nil {
		e.telemetry.SetQueueGetter(func() (int, int, int) {
			return e.scheduler.FrontierSize(), len(e.fetchCh), len(e.parseCh)
		})
		e.telemetry.SetHostGetter(func() map[string]metrics.HostSnapshot {
			snapshot := e.scheduler.HostStatesSnapshot()
			out := make(map[string]metrics.HostSnapshot, len(snapshot))
			for host, hs := range snapshot {
				out[host] = metrics.HostSnapshot{Inflight: hs.Semaphore.Inflight(), Circuit: string(hs.State())}
			}
			return out
		})
		e.telemetry.SetRobotsManager(e.robotsMgr)
		go e.telemetry.Run(e.ctx)
	}

	go e.scheduler.Run()
	go e.storageLoop()
	go e.monitorStop()

	fetchWorkers := max(4, e.cfg.GlobalConcurrency)
	parseWorkers := max(2, e.cfg.GlobalConcurrency/2)
	for i := 0; i < fetchWorkers; i++ {
		go e.fetchLoop()
	}
	for i := 0; i < parseWorkers; i++ {
		go e.parseLoop()
	}

	e.enqueueURL(seed, 0, "")
	if e.cfg.TimeBudget > 0 {
		go e.stopAfterBudget()
	}
}

func (e *Engine) monitorStop() {
	<-e.ctx.Done()
	now := time.Now()
	_ = e.store.UpdateRunStatus(context.Background(), e.runID, "stopped", nil, &now)
}

func (e *Engine) Stop() {
	e.stopOnce.Do(func() {
		e.cancel()
	})
}

func (e *Engine) PagesFetched() int64 {
	return e.pagesFetched.Load()
}

func (e *Engine) Done() <-chan struct{} {
	return e.ctx.Done()
}

func (e *Engine) stopAfterBudget() {
	t := time.NewTimer(e.cfg.TimeBudget)
	defer t.Stop()
	select {
	case <-e.ctx.Done():
		return
	case <-t.C:
		e.Stop()
	}
}

func (e *Engine) enqueueURL(raw string, depth int, sourceHost string) {
	if e.ctx.Err() != nil {
		return
	}
	canonical, parsed, err := Canonicalize(raw)
	if err != nil {
		return
	}
	if e.deduper.Seen(canonical) {
		return
	}
	host := HostKey(parsed)
	task := &Task{URL: parsed.String(), Canonical: canonical, Host: host, Depth: depth, SourceHost: sourceHost, DiscoveredAt: time.Now()}
	select {
	case e.enqueueCh <- task:
		// ok
	default:
		// backpressure: block until space or context done
		select {
		case e.enqueueCh <- task:
		case <-e.ctx.Done():
		}
	}
}

func (e *Engine) fetchLoop() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case task := <-e.fetchCh:
			if task == nil {
				continue
			}
			e.handleFetch(task)
		}
	}
}

func (e *Engine) handleFetch(task *Task) {
	defer task.Permit.Release()

	if e.cfg.MaxPages > 0 && int(e.pagesFetched.Load()) >= e.cfg.MaxPages {
		return
	}

	ctx, cancel := context.WithTimeout(e.ctx, e.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		e.recordError(task, ErrFetch, err.Error())
		return
	}
	if e.cfg.UserAgent != "" {
		req.Header.Set("User-Agent", e.cfg.UserAgent)
	}

	var reusedConn bool
	trace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			reusedConn = info.Reused
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	start := time.Now()
	resp, err := e.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		class := classifyError(err)
		if e.shouldRetry(task, class, 0) {
			return
		}
		e.recordFetch(task, 0, "", nil, latency, 0, reusedConn, class, err.Error())
		return
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	contentType := resp.Header.Get("Content-Type")

	if status >= 300 && status < 400 {
		location := resp.Header.Get("Location")
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		e.handleRedirect(task, location)
		e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, "", "")
		return
	}

	if status == http.StatusTooManyRequests {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		if e.shouldRetry(task, ErrStatus, retryAfter) {
			return
		}
		e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrStatus, "too_many_requests")
		return
	}

	if status >= 500 {
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		if e.shouldRetry(task, ErrStatus, 0) {
			return
		}
		e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrStatus, resp.Status)
		return
	}

	if status >= 400 {
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrStatus, resp.Status)
		return
	}

	needBody := isHTML(contentType) && (e.cfg.MaxDepth <= 0 || task.Depth < e.cfg.MaxDepth)
	if needBody {
		body, size, errClass := readBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		if errClass == ErrSizeLimit {
			e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrSizeLimit, "max_body_bytes")
			return
		}
		if errClass != "" {
			e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrFetch, errClass)
			return
		}
		e.recordFetch(task, status, contentType, body, latency, size, reusedConn, "", "")
		return
	}

	size, errClass := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
	if errClass == ErrSizeLimit {
		e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrSizeLimit, "max_body_bytes")
		return
	}
	if errClass != "" {
		e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, ErrFetch, errClass)
		return
	}
	e.recordFetch(task, status, contentType, nil, latency, size, reusedConn, "", "")
}

func (e *Engine) recordFetch(task *Task, status int, contentType string, body []byte, latency int64, size int64, reused bool, errClass, errMessage string) {
	if errClass == "" {
		e.pagesFetched.Add(1)
		metrics.PagesFetched.Inc()
	} else {
		metrics.FetchErrors.WithLabelValues(errClass).Inc()
	}

	if e.telemetry != nil {
		select {
		case e.telemetry.FetchEvents() <- metrics.FetchEvent{Host: task.Host, LatencyMS: latency, Bytes: size, ReusedConn: reused, ErrClass: errClass}:
		default:
		}
	}

	success := errClass == "" && status < 500
	if hs := e.scheduler.HostState(task.Host); hs != nil {
		hs.OnResult(success)
	}

	if e.cfg.MaxPages > 0 && int(e.pagesFetched.Load()) >= e.cfg.MaxPages {
		e.Stop()
	}

	fetchedAt := time.Now()
	discovered := task.DiscoveredAt
	if discovered.IsZero() {
		discovered = e.startedAt
	}
	rec := storage.PageRecord{
		RunID:        e.runID,
		URL:          task.URL,
		CanonicalURL: task.Canonical,
		Host:         task.Host,
		Depth:        task.Depth,
		StatusCode:   status,
		ContentType:  contentType,
		FetchMS:      latency,
		SizeBytes:    size,
		ErrClass:     errClass,
		ErrMessage:   errMessage,
		DiscoveredAt: discovered,
		FetchedAt:    &fetchedAt,
	}
	select {
	case e.pageWrites <- rec:
	default:
	}

	if errClass != "" {
		select {
		case e.errorWrites <- errorRecord{runID: e.runID, host: task.Host, url: task.URL, class: errClass, message: errMessage}:
		default:
		}
	}

	if body != nil && isHTML(contentType) && (e.cfg.MaxDepth <= 0 || task.Depth < e.cfg.MaxDepth) {
		select {
		case e.parseCh <- &FetchResult{Task: task, StatusCode: status, ContentType: contentType, Body: body, FetchMS: latency, SizeBytes: size, ReusedConn: reused}:
		default:
			// drop parse if backpressure
		}
	}
}

func (e *Engine) handleRedirect(task *Task, location string) {
	if location == "" {
		return
	}
	base, err := url.Parse(task.URL)
	if err != nil {
		return
	}
	loc, err := url.Parse(location)
	if err != nil {
		return
	}
	resolved := base.ResolveReference(loc)
	canonical, parsed, err := Canonicalize(resolved.String())
	if err != nil {
		return
	}
	if e.deduper.Seen(canonical) {
		return
	}
	host := HostKey(parsed)
	task.SourceHost = task.Host
	newTask := &Task{URL: resolved.String(), Canonical: canonical, Host: host, Depth: task.Depth, SourceHost: task.Host, DiscoveredAt: time.Now()}
	select {
	case e.enqueueCh <- newTask:
	default:
		select {
		case e.enqueueCh <- newTask:
		case <-e.ctx.Done():
		}
	}
	if task.Host != host && e.telemetry != nil {
		select {
		case e.telemetry.EdgeEvents() <- metrics.EdgeEvent{Src: task.Host, Dst: host}:
		default:
		}
	}
	select {
	case e.edgeWrites <- edgeRecord{runID: e.runID, src: task.Host, dst: host, count: 1}:
	default:
	}
}

func (e *Engine) parseLoop() {
	for {
		select {
		case <-e.ctx.Done():
			return
		case res := <-e.parseCh:
			if res == nil {
				continue
			}
			e.handleParse(res)
		}
	}
}

func (e *Engine) handleParse(res *FetchResult) {
	if e.cfg.MaxDepth > 0 && res.Task.Depth >= e.cfg.MaxDepth {
		return
	}
	baseURL, err := url.Parse(res.Task.URL)
	if err != nil {
		return
	}

	tok := html.NewTokenizer(bytes.NewReader(res.Body))
	linksFound := 0
	for {
		tt := tok.Next()
		switch tt {
		case html.ErrorToken:
			if tok.Err() != io.EOF {
				e.recordError(res.Task, ErrParse, tok.Err().Error())
			}
			return
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := tok.TagName()
			if string(name) != "a" || !hasAttr {
				continue
			}
			for {
				key, val, more := tok.TagAttr()
				if string(key) == "href" {
					link := strings.TrimSpace(string(val))
						if link != "" {
							if strings.HasPrefix(link, "//") {
								link = baseURL.Scheme + ":" + link
							}
							parsedLink, err := url.Parse(link)
							if err == nil {
							resolved := baseURL.ResolveReference(parsedLink)
							canonical, parsed, err := Canonicalize(resolved.String())
							if err == nil && !e.deduper.Seen(canonical) {
								host := HostKey(parsed)
								task := &Task{URL: resolved.String(), Canonical: canonical, Host: host, Depth: res.Task.Depth + 1, SourceHost: res.Task.Host, DiscoveredAt: time.Now()}
								select {
								case e.enqueueCh <- task:
								default:
									select {
									case e.enqueueCh <- task:
									case <-e.ctx.Done():
										return
									}
								}
								if res.Task.Host != host && e.telemetry != nil {
									select {
									case e.telemetry.EdgeEvents() <- metrics.EdgeEvent{Src: res.Task.Host, Dst: host}:
									default:
									}
								}
								select {
								case e.edgeWrites <- edgeRecord{runID: e.runID, src: res.Task.Host, dst: host, count: 1}:
								default:
								}
								linksFound++
								if e.cfg.MaxLinksPerPage > 0 && linksFound >= e.cfg.MaxLinksPerPage {
									return
								}
							}
						}
					}
				}
				if !more {
					break
				}
			}
		}
	}
}

func (e *Engine) recordError(task *Task, class, message string) {
	if e.telemetry != nil {
		select {
		case e.telemetry.FetchEvents() <- metrics.FetchEvent{Host: task.Host, ErrClass: class}:
		default:
		}
	}
	select {
	case e.errorWrites <- errorRecord{runID: e.runID, host: task.Host, url: task.URL, class: class, message: message}:
	default:
	}
}

func (e *Engine) storageLoop() {
	ctx := context.Background()
	for {
		select {
		case <-e.ctx.Done():
			return
		case rec := <-e.pageWrites:
			if err := e.store.InsertPage(ctx, rec); err != nil {
				log.Printf("store page: %v", err)
			}
		case rec := <-e.errorWrites:
			if err := e.store.InsertError(ctx, rec.runID, rec.host, rec.url, rec.class, rec.message); err != nil {
				log.Printf("store error: %v", err)
			}
		case rec := <-e.edgeWrites:
			if err := e.store.UpsertEdge(ctx, rec.runID, rec.src, rec.dst, rec.count); err != nil {
				log.Printf("store edge: %v", err)
			}
		}
	}
}

func (e *Engine) shouldRetry(task *Task, class string, retryAfter time.Duration) bool {
	if task.Retries >= e.cfg.RetryMax {
		return false
	}
	if class == ErrStatus || class == ErrTimeout || class == ErrTLS || class == ErrDNS || class == ErrFetch {
		task.Retries++
		delay := e.cfg.RetryBaseDelay * time.Duration(1<<task.Retries)
		if retryAfter > 0 {
			delay = retryAfter
		}
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		task.NotBefore = time.Now().Add(delay)
		select {
		case e.enqueueCh <- task:
			return true
		default:
			select {
			case e.enqueueCh <- task:
				return true
			case <-e.ctx.Done():
				return false
			}
		}
	}
	return false
}

func classifyError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrTimeout
		}
	}
	if strings.Contains(err.Error(), "tls") {
		return ErrTLS
	}
	if strings.Contains(err.Error(), "no such host") {
		return ErrDNS
	}
	return ErrFetch
}

func readBodyLimited(r io.Reader, max int64) ([]byte, int64, string) {
	lr := &io.LimitedReader{R: r, N: max + 1}
	data, err := io.ReadAll(lr)
	size := int64(len(data))
	if size > max {
		return nil, size, ErrSizeLimit
	}
	if err != nil {
		return nil, size, err.Error()
	}
	return data, size, ""
}

func drainBodyLimited(r io.Reader, max int64) (int64, string) {
	n, err := io.CopyN(io.Discard, r, max+1)
	if n > max {
		return n, ErrSizeLimit
	}
	if err != nil && err != io.EOF {
		return n, err.Error()
	}
	return n, ""
}

func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "text/html") || strings.HasPrefix(ct, "application/xhtml+xml")
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if n, err := strconv.Atoi(value); err == nil {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package crawler

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"webcrawler/internal/crawler/robots"
)

type EngineHooks struct {
	OnPage     func(Page)
	OnTreeNode func(TreeNode)
	OnTreeEdge func(TreeEdge)
	OnSkip     func(DiagnosticEntry)
	OnRetry    func(DiagnosticEntry)
	OnError    func(DiagnosticEntry)
	OnFetch    func(DiagnosticEntry)
	OnComplete func(string)
}

type Engine struct {
	runID     string
	cfg       RunConfig
	sourceURL string
	hooks     EngineHooks

	ctx    context.Context
	cancel context.CancelFunc

	deduper   *Deduper
	scheduler *Scheduler
	robotsMgr *robots.Manager
	client    *http.Client

	enqueueCh chan *Task
	fetchCh   chan *Task

	pagesFetched atomic.Int64
	stopReasonMu sync.Mutex
	stopReason   string
	stopOnce     sync.Once
}

const (
	StopReasonManual     = "manual"
	StopReasonMaxPages   = "max_pages"
	StopReasonTimeBudget = "time_budget"
	StopReasonUnknown    = "unknown"
)

func NewEngine(runID string, cfg RunConfig, hooks EngineHooks) *Engine {
	cfg = cfg.Normalize()
	if cfg.GlobalConcurrency <= 0 {
		cfg.GlobalConcurrency = 16
	}
	if cfg.PerHostConcurrency <= 0 {
		cfg.PerHostConcurrency = 4
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 2 << 20
	}
	if cfg.RetryMax < 0 {
		cfg.RetryMax = 0
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalSem := NewSemaphore(cfg.GlobalConcurrency)
	enqueueCh := make(chan *Task, cfg.GlobalConcurrency*16)
	fetchCh := make(chan *Task, cfg.GlobalConcurrency*2)
	client := &http.Client{
		Transport: buildTransport(cfg),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	var robotsMgr *robots.Manager
	var scheduler *Scheduler
	if cfg.RespectRobots {
		rm := robots.New(client, cfg.UserAgent, cfg.RobotsTTL, 4)
		robotsMgr = rm
		scheduler = NewScheduler(ctx, enqueueCh, fetchCh, 0, globalSem, cfg.PerHostConcurrency, cfg.CircuitTripCount, cfg.CircuitResetTime, true, rm, func(task *Task, reason string) {
			if hooks.OnSkip != nil && task != nil {
				hooks.OnSkip(DiagnosticEntry{
					At:       time.Now(),
					URL:      task.URL,
					PageID:   task.PageID,
					ParentID: task.ParentPageID,
					Reason:   reason,
				})
			}
		})
	} else {
		scheduler = NewScheduler(ctx, enqueueCh, fetchCh, 0, globalSem, cfg.PerHostConcurrency, cfg.CircuitTripCount, cfg.CircuitResetTime, false, nil, func(task *Task, reason string) {
			if hooks.OnSkip != nil && task != nil {
				hooks.OnSkip(DiagnosticEntry{
					At:       time.Now(),
					URL:      task.URL,
					PageID:   task.PageID,
					ParentID: task.ParentPageID,
					Reason:   reason,
				})
			}
		})
	}

	return &Engine{
		runID:     runID,
		cfg:       cfg,
		sourceURL: cfg.SeedURL,
		hooks:     hooks,
		ctx:       ctx,
		cancel:    cancel,
		deduper:   NewDeduper(64),
		scheduler: scheduler,
		robotsMgr: robotsMgr,
		client:    client,
		enqueueCh: enqueueCh,
		fetchCh:   fetchCh,
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

func (e *Engine) Start(rootPageID string, prefetchedRoot *Page) {
	go e.scheduler.Run()
	for i := 0; i < max(4, e.cfg.GlobalConcurrency); i++ {
		go e.fetchLoop()
	}
	rootCanonical, parsed, err := Canonicalize(e.sourceURL)
	if err != nil {
		e.StopWithReason("invalid_seed")
		return
	}
	e.deduper.Seen(rootCanonical)
	rootNode := TreeNode{
		ID:    rootPageID,
		URL:   parsed.String(),
		Title: parsed.Host,
		Depth: 0,
	}
	if e.hooks.OnTreeNode != nil {
		e.hooks.OnTreeNode(rootNode)
	}
	if prefetchedRoot != nil && prefetchedRoot.CanonicalURL == rootCanonical {
		e.emitPrefetchedRoot(rootPageID, *prefetchedRoot)
		if e.cfg.TimeBudget > 0 {
			go e.stopAfterBudget()
		}
		return
	}
	e.enqueueTask(&Task{
		PageID:       rootPageID,
		ParentPageID: "",
		URL:          parsed.String(),
		Canonical:    rootCanonical,
		Host:         HostKey(parsed),
		Depth:        0,
		DiscoveredAt: time.Now(),
	})
	if e.cfg.TimeBudget > 0 {
		go e.stopAfterBudget()
	}
}

func (e *Engine) emitPrefetchedRoot(rootPageID string, page Page) {
	now := time.Now()
	page.ID = rootPageID
	page.ParentPageID = ""
	page.SourceMode = string(e.cfg.Mode)
	page.SourceInput = e.cfg.Input
	page.Depth = 0
	page.DiscoveredAt = now
	if page.FetchedAt.IsZero() {
		page.FetchedAt = now
	}
	e.pagesFetched.Add(1)
	if e.hooks.OnPage != nil {
		e.hooks.OnPage(page)
	}
	if e.cfg.MaxPages > 0 && int(e.pagesFetched.Load()) >= e.cfg.MaxPages {
		e.StopWithReason(StopReasonMaxPages)
		return
	}
	for _, link := range page.OutgoingLinks {
		childCanonical, parsed, err := Canonicalize(link)
		if err != nil {
			e.emitSkip(nil, link, "invalid_link")
			continue
		}
		if e.deduper.Seen(childCanonical) {
			e.emitSkip(nil, parsed.String(), "duplicate")
			continue
		}
		childID := NewPageID()
		childTask := &Task{
			PageID:       childID,
			ParentPageID: rootPageID,
			URL:          parsed.String(),
			Canonical:    childCanonical,
			Host:         HostKey(parsed),
			Depth:        1,
			DiscoveredAt: time.Now(),
		}
		if e.hooks.OnTreeNode != nil {
			e.hooks.OnTreeNode(TreeNode{
				ID:           childID,
				ParentPageID: rootPageID,
				URL:          parsed.String(),
				Title:        parsed.Host,
				Depth:        childTask.Depth,
			})
		}
		if e.hooks.OnTreeEdge != nil {
			e.hooks.OnTreeEdge(TreeEdge{ParentPageID: rootPageID, ChildPageID: childID})
		}
		e.enqueueTask(childTask)
	}
}

func (e *Engine) Stop() {
	e.StopWithReason(StopReasonManual)
}

func (e *Engine) StopWithReason(reason string) {
	if reason == "" {
		reason = StopReasonManual
	}
	e.setStopReason(reason)
	e.stopOnce.Do(func() {
		e.cancel()
		if e.hooks.OnComplete != nil {
			e.hooks.OnComplete(reason)
		}
	})
}

func (e *Engine) Done() <-chan struct{} {
	return e.ctx.Done()
}

func (e *Engine) PagesFetched() int64 {
	return e.pagesFetched.Load()
}

func (e *Engine) StopReason() string {
	e.stopReasonMu.Lock()
	defer e.stopReasonMu.Unlock()
	return e.stopReason
}

func (e *Engine) setStopReason(reason string) {
	e.stopReasonMu.Lock()
	defer e.stopReasonMu.Unlock()
	if e.stopReason == "" {
		e.stopReason = reason
	}
}

func (e *Engine) stopAfterBudget() {
	timer := time.NewTimer(e.cfg.TimeBudget)
	defer timer.Stop()
	select {
	case <-e.ctx.Done():
	case <-timer.C:
		e.StopWithReason(StopReasonTimeBudget)
	}
}

func (e *Engine) enqueueTask(task *Task) {
	if task == nil || e.ctx.Err() != nil {
		return
	}
	select {
	case e.enqueueCh <- task:
	default:
		if e.hooks.OnSkip != nil {
			e.hooks.OnSkip(DiagnosticEntry{
				At:       time.Now(),
				URL:      task.URL,
				PageID:   task.PageID,
				ParentID: task.ParentPageID,
				Reason:   "enqueue_buffer_full",
			})
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
		e.StopWithReason(StopReasonMaxPages)
		return
	}

	ctx, cancel := context.WithTimeout(e.ctx, e.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
	if err != nil {
		e.recordFailure(task, 0, "", 0, 0, ErrFetch, err.Error())
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
		e.recordFailure(task, 0, "", latency, 0, class, err.Error())
		return
	}
	defer resp.Body.Close()

	status := resp.StatusCode
	contentType := resp.Header.Get("Content-Type")
	if status >= 300 && status < 400 {
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		location := resp.Header.Get("Location")
		e.recordFetch(task, reusedConn, status, latency, size)
		e.handleRedirect(task, location)
		return
	}

	if status == http.StatusTooManyRequests {
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		if e.shouldRetry(task, ErrStatus, retryAfter) {
			return
		}
		e.recordFailure(task, status, contentType, latency, size, ErrStatus, "too_many_requests")
		return
	}

	if status >= 400 {
		size, _ := drainBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
		if status >= 500 && e.shouldRetry(task, ErrStatus, 0) {
			return
		}
		e.recordFailure(task, status, contentType, latency, size, ErrStatus, resp.Status)
		return
	}

	body, size, errClass := readBodyLimited(resp.Body, e.cfg.MaxBodyBytes)
	if errClass != "" {
		e.recordFailure(task, status, contentType, latency, size, ErrFetch, errClass)
		return
	}
	e.recordFetch(task, reusedConn, status, latency, size)

	extracted := ExtractContent(mustParseURL(task.URL), body, e.cfg.MaxLinksPerPage)
	fetchedAt := time.Now()
	page := Page{
		ID:            task.PageID,
		ParentPageID:  task.ParentPageID,
		SourceMode:    string(e.cfg.Mode),
		SourceInput:   e.cfg.Input,
		URL:           task.URL,
		CanonicalURL:  task.Canonical,
		Host:          task.Host,
		Title:         chooseTitle(extracted.Title, task.URL),
		Text:          extracted.Text,
		Excerpt:       buildExcerpt(extracted.Text),
		OutgoingLinks: extracted.Links,
		Depth:         task.Depth,
		StatusCode:    status,
		ContentType:   contentType,
		FetchMS:       latency,
		SizeBytes:     size,
		DiscoveredAt:  task.DiscoveredAt,
		FetchedAt:     fetchedAt,
	}
	e.pagesFetched.Add(1)
	if e.hooks.OnPage != nil {
		e.hooks.OnPage(page)
	}
	if e.cfg.MaxPages > 0 && int(e.pagesFetched.Load()) >= e.cfg.MaxPages {
		e.StopWithReason(StopReasonMaxPages)
	}
	for _, link := range extracted.Links {
		childCanonical, parsed, err := Canonicalize(link)
		if err != nil {
			e.emitSkip(task, link, "invalid_link")
			continue
		}
		if e.deduper.Seen(childCanonical) {
			e.emitSkip(task, parsed.String(), "duplicate")
			continue
		}
		childID := NewPageID()
		childTask := &Task{
			PageID:       childID,
			ParentPageID: task.PageID,
			URL:          parsed.String(),
			Canonical:    childCanonical,
			Host:         HostKey(parsed),
			Depth:        task.Depth + 1,
			DiscoveredAt: time.Now(),
		}
		if e.hooks.OnTreeNode != nil {
			e.hooks.OnTreeNode(TreeNode{
				ID:           childID,
				ParentPageID: task.PageID,
				URL:          parsed.String(),
				Title:        parsed.Host,
				Depth:        childTask.Depth,
			})
		}
		if e.hooks.OnTreeEdge != nil {
			e.hooks.OnTreeEdge(TreeEdge{ParentPageID: task.PageID, ChildPageID: childID})
		}
		e.enqueueTask(childTask)
	}
}

func (e *Engine) recordFetch(task *Task, reused bool, status int, latency, size int64) {
	if e.hooks.OnFetch != nil {
		e.hooks.OnFetch(DiagnosticEntry{
			At:     time.Now(),
			URL:    task.URL,
			PageID: task.PageID,
			Reason: "fetch_complete",
			Detail: "status=" + strconv.Itoa(status) + ",reused=" + strconv.FormatBool(reused) + ",latency_ms=" + strconv.FormatInt(latency, 10) + ",bytes=" + strconv.FormatInt(size, 10),
		})
	}
	if hs := e.scheduler.HostState(task.Host); hs != nil {
		hs.OnResult(true)
	}
}

func (e *Engine) recordFailure(task *Task, status int, contentType string, latency, size int64, class, message string) {
	if hs := e.scheduler.HostState(task.Host); hs != nil {
		hs.OnResult(false)
	}
	page := Page{
		ID:           task.PageID,
		ParentPageID: task.ParentPageID,
		SourceMode:   string(e.cfg.Mode),
		SourceInput:  e.cfg.Input,
		URL:          task.URL,
		CanonicalURL: task.Canonical,
		Host:         task.Host,
		Depth:        task.Depth,
		StatusCode:   status,
		ContentType:  contentType,
		FetchMS:      latency,
		SizeBytes:    size,
		ErrorClass:   class,
		ErrorMessage: message,
		DiscoveredAt: task.DiscoveredAt,
		FetchedAt:    time.Now(),
	}
	if e.hooks.OnPage != nil {
		e.hooks.OnPage(page)
	}
	if e.hooks.OnError != nil {
		e.hooks.OnError(DiagnosticEntry{
			At:       time.Now(),
			URL:      task.URL,
			PageID:   task.PageID,
			ParentID: task.ParentPageID,
			Reason:   class,
			Detail:   message,
		})
	}
}

func (e *Engine) emitSkip(parent *Task, rawURL, reason string) {
	if e.hooks.OnSkip != nil {
		entry := DiagnosticEntry{At: time.Now(), URL: rawURL, Reason: reason}
		if parent != nil {
			entry.ParentID = parent.PageID
		}
		e.hooks.OnSkip(entry)
	}
}

func (e *Engine) handleRedirect(task *Task, location string) {
	if location == "" {
		e.emitSkip(task, task.URL, "redirect_without_location")
		return
	}
	base, err := url.Parse(task.URL)
	if err != nil {
		e.emitSkip(task, task.URL, "invalid_redirect_base")
		return
	}
	loc, err := url.Parse(location)
	if err != nil {
		e.emitSkip(task, location, "invalid_redirect_target")
		return
	}
	resolved := base.ResolveReference(loc)
	canonical, parsed, err := Canonicalize(resolved.String())
	if err != nil {
		e.emitSkip(task, resolved.String(), "invalid_redirect_target")
		return
	}
	if e.deduper.Seen(canonical) {
		e.emitSkip(task, parsed.String(), "duplicate_redirect_target")
		return
	}
	redirectID := NewPageID()
	if e.hooks.OnTreeNode != nil {
		e.hooks.OnTreeNode(TreeNode{
			ID:           redirectID,
			ParentPageID: task.PageID,
			URL:          parsed.String(),
			Title:        parsed.Host,
			Depth:        task.Depth,
		})
	}
	if e.hooks.OnTreeEdge != nil {
		e.hooks.OnTreeEdge(TreeEdge{ParentPageID: task.PageID, ChildPageID: redirectID})
	}
	e.enqueueTask(&Task{
		PageID:       redirectID,
		ParentPageID: task.PageID,
		URL:          parsed.String(),
		Canonical:    canonical,
		Host:         HostKey(parsed),
		Depth:        task.Depth,
		DiscoveredAt: time.Now(),
	})
}

func (e *Engine) shouldRetry(task *Task, class string, retryAfter time.Duration) bool {
	if task.Retries >= e.cfg.RetryMax {
		return false
	}
	if class != ErrStatus && class != ErrTimeout && class != ErrTLS && class != ErrDNS && class != ErrFetch {
		return false
	}
	task.Retries++
	delay := e.cfg.RetryBaseDelay * time.Duration(1<<task.Retries)
	if retryAfter > 0 {
		delay = retryAfter
	}
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}
	task.NotBefore = time.Now().Add(delay)
	if e.hooks.OnRetry != nil {
		e.hooks.OnRetry(DiagnosticEntry{
			At:       time.Now(),
			URL:      task.URL,
			PageID:   task.PageID,
			ParentID: task.ParentPageID,
			Reason:   class,
			Detail:   "retry_at=" + task.NotBefore.Format(time.RFC3339),
		})
	}
	e.enqueueTask(task)
	return true
}

func classifyError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrTimeout
	}
	if strings.Contains(strings.ToLower(err.Error()), "tls") {
		return ErrTLS
	}
	if strings.Contains(strings.ToLower(err.Error()), "no such host") {
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

func chooseTitle(title, fallback string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	u, err := url.Parse(fallback)
	if err != nil {
		return fallback
	}
	return u.Host
}

func mustParseURL(raw string) *url.URL {
	u, _ := url.Parse(raw)
	return u
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

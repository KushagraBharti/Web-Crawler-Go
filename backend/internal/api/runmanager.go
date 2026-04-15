package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"webcrawler/internal/artifacts"
	"webcrawler/internal/config"
	"webcrawler/internal/crawler"
	"webcrawler/internal/search"
)

type RunState struct {
	ID         string
	Config     crawler.RunConfig
	Seed       crawler.SearchSeed
	Status     string
	StopReason string
	CreatedAt  time.Time
	StartedAt  *time.Time
	StoppedAt  *time.Time
	Engine     *crawler.Engine

	pagesByID   map[string]crawler.Page
	pageOrder   []string
	treeNodes   map[string]crawler.TreeNode
	treeEdges   []crawler.TreeEdge
	diagnostics crawler.Diagnostics
	subscribers map[int]chan crawler.EventFrame
	nextSubID   int
}

type CreateRunInput struct {
	Mode               crawler.SourceMode `json:"mode"`
	Input              string             `json:"input"`
	MaxDepth           int                `json:"max_depth"`
	MaxPages           int                `json:"max_pages"`
	TimeBudgetSeconds  int                `json:"time_budget_seconds"`
	MaxLinksPerPage    int                `json:"max_links_per_page"`
	GlobalConcurrency  int                `json:"global_concurrency"`
	PerHostConcurrency int                `json:"per_host_concurrency"`
	UserAgent          string             `json:"user_agent"`
	RespectRobots      *bool              `json:"respect_robots"`
}

type RunManager struct {
	defaults  config.CrawlerDefaults
	resolver  *search.Resolver
	artifacts *artifacts.Store

	mu   sync.Mutex
	runs map[string]*RunState
}

func NewRunManager(defaults config.CrawlerDefaults, artifactRoot string, searchBaseURL string, searchAPIKey string) *RunManager {
	httpClient := &http.Client{Timeout: 15 * time.Second}
	return &RunManager{
		defaults:  defaults,
		resolver:  search.New(searchBaseURL, searchAPIKey, httpClient, defaults.UserAgent),
		artifacts: artifacts.New(artifactRoot),
		runs:      make(map[string]*RunState),
	}
}

func (rm *RunManager) CreateRun(ctx context.Context, input CreateRunInput) (*RunState, error) {
	cfg := rm.applyDefaults(crawler.RunConfig{
		Mode:               input.Mode,
		Input:              input.Input,
		MaxDepth:           input.MaxDepth,
		MaxPages:           input.MaxPages,
		TimeBudgetSeconds:  input.TimeBudgetSeconds,
		MaxLinksPerPage:    input.MaxLinksPerPage,
		GlobalConcurrency:  input.GlobalConcurrency,
		PerHostConcurrency: input.PerHostConcurrency,
		UserAgent:          input.UserAgent,
	})
	if input.RespectRobots != nil {
		cfg.RespectRobots = *input.RespectRobots
	}
	cfg = cfg.Normalize()

	runID := uuid.NewString()
	seed, searchLog, err := rm.resolveSeed(ctx, cfg)
	if err != nil {
		log.Printf("create run failed mode=%s input=%q err=%v search_attempts=%s", cfg.Mode, cfg.Input, err, formatSearchLog(searchLog))
		return nil, err
	}
	cfg.SeedURL = seed.PrimaryURL
	if cfg.Mode == crawler.SourceModeKeyword {
		log.Printf("resolved keyword seed query=%q primary=%s search_attempts=%s", cfg.Input, seed.PrimaryURL, formatSearchLog(searchLog))
	}

	run := &RunState{
		ID:          runID,
		Config:      cfg,
		Seed:        seed,
		Status:      "created",
		CreatedAt:   time.Now().UTC(),
		pagesByID:   make(map[string]crawler.Page),
		treeNodes:   make(map[string]crawler.TreeNode),
		subscribers: make(map[int]chan crawler.EventFrame),
		diagnostics: crawler.Diagnostics{
			SearchSeed:    &seed,
			SearchLog:     searchLog,
			SkippedURLs:   []crawler.DiagnosticEntry{},
			RetryEvents:   []crawler.DiagnosticEntry{},
			Errors:        []crawler.DiagnosticEntry{},
			FetchLog:      []crawler.DiagnosticEntry{},
			ArtifactDir:   rm.artifacts.RunDir(runID),
			ArtifactFiles: rm.artifacts.Paths(runID),
		},
	}
	rm.mu.Lock()
	rm.runs[runID] = run
	rm.mu.Unlock()
	if err := rm.persistLocked(run); err != nil {
		return nil, err
	}
	return rm.cloneRun(run), nil
}

func (rm *RunManager) StartRun(ctx context.Context, runID string) error {
	rm.mu.Lock()
	run, ok := rm.runs[runID]
	if !ok {
		rm.mu.Unlock()
		return fmt.Errorf("run not found")
	}
	if run.Engine != nil {
		rm.mu.Unlock()
		return fmt.Errorf("run already started")
	}
	now := time.Now().UTC()
	run.Status = "running"
	run.StartedAt = &now
	rootID := crawler.NewPageID()
	engine := crawler.NewEngine(runID, run.Config, crawler.EngineHooks{
		OnPage: func(page crawler.Page) {
			rm.onPage(runID, page)
		},
		OnTreeNode: func(node crawler.TreeNode) {
			rm.onTreeNode(runID, node)
		},
		OnTreeEdge: func(edge crawler.TreeEdge) {
			rm.onTreeEdge(runID, edge)
		},
		OnSkip: func(entry crawler.DiagnosticEntry) {
			rm.onDiagnostic(runID, "skip", entry)
		},
		OnRetry: func(entry crawler.DiagnosticEntry) {
			rm.onDiagnostic(runID, "retry", entry)
		},
		OnError: func(entry crawler.DiagnosticEntry) {
			rm.onDiagnostic(runID, "error", entry)
		},
		OnFetch: func(entry crawler.DiagnosticEntry) {
			rm.onDiagnostic(runID, "fetch", entry)
		},
		OnComplete: func(reason string) {
			rm.onComplete(runID, reason)
		},
	})
	run.Engine = engine
	rm.broadcastLocked(run, crawler.EventFrame{
		Ts:      time.Now().UTC(),
		Status:  run.Status,
		Summary: rm.summaryLocked(run),
	})
	if err := rm.persistLocked(run); err != nil {
		rm.mu.Unlock()
		return err
	}
	rm.mu.Unlock()

	go engine.Start(rootID)
	return nil
}

func (rm *RunManager) StopRun(runID string) error {
	rm.mu.Lock()
	run, ok := rm.runs[runID]
	rm.mu.Unlock()
	if !ok {
		return fmt.Errorf("run not found")
	}
	if run.Engine != nil {
		run.Engine.Stop()
	}
	return nil
}

func (rm *RunManager) ListRuns() []crawler.Snapshot {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	out := make([]crawler.Snapshot, 0, len(rm.runs))
	for _, run := range rm.runs {
		out = append(out, rm.snapshotLocked(run))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

func (rm *RunManager) GetRun(runID string) (crawler.Snapshot, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return crawler.Snapshot{}, fmt.Errorf("run not found")
	}
	return rm.snapshotLocked(run), nil
}

func (rm *RunManager) ListPages(runID string) ([]crawler.Page, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return nil, fmt.Errorf("run not found")
	}
	return rm.pagesLocked(run), nil
}

func (rm *RunManager) GetPage(runID, pageID string) (crawler.Page, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return crawler.Page{}, fmt.Errorf("run not found")
	}
	page, ok := run.pagesByID[pageID]
	if !ok {
		return crawler.Page{}, fmt.Errorf("page not found")
	}
	return page, nil
}

func (rm *RunManager) GetTree(runID string) (map[string]any, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return nil, fmt.Errorf("run not found")
	}
	return map[string]any{
		"nodes": rm.treeNodesLocked(run),
		"edges": append([]crawler.TreeEdge(nil), run.treeEdges...),
	}, nil
}

func (rm *RunManager) GetDiagnostics(runID string) (crawler.Diagnostics, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return crawler.Diagnostics{}, fmt.Errorf("run not found")
	}
	return run.diagnostics, nil
}

func (rm *RunManager) DeleteRun(runID string) error {
	rm.mu.Lock()
	run, ok := rm.runs[runID]
	if !ok {
		rm.mu.Unlock()
		return fmt.Errorf("run not found")
	}
	if run.Status == "running" {
		rm.mu.Unlock()
		return fmt.Errorf("cannot delete running run")
	}
	delete(rm.runs, runID)
	rm.mu.Unlock()
	return os.RemoveAll(rm.artifacts.RunDir(runID))
}

func (rm *RunManager) Subscribe(runID string) (<-chan crawler.EventFrame, func(), error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return nil, nil, fmt.Errorf("run not found")
	}
	id := run.nextSubID
	run.nextSubID++
	ch := make(chan crawler.EventFrame, 16)
	run.subscribers[id] = ch
	ch <- crawler.EventFrame{Ts: time.Now().UTC(), Status: run.Status, Summary: rm.summaryLocked(run)}
	return ch, func() {
		rm.mu.Lock()
		defer rm.mu.Unlock()
		if c, ok := run.subscribers[id]; ok {
			delete(run.subscribers, id)
			close(c)
		}
	}, nil
}

func (rm *RunManager) onPage(runID string, page crawler.Page) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return
	}
	_, exists := run.pagesByID[page.ID]
	run.pagesByID[page.ID] = page
	if !exists {
		run.pageOrder = append(run.pageOrder, page.ID)
	}
	frame := crawler.EventFrame{
		Ts:       time.Now().UTC(),
		Status:   run.Status,
		Summary:  rm.summaryLocked(run),
		NewPages: []crawler.Page{page},
		Latest:   &page,
	}
	rm.broadcastLocked(run, frame)
	_ = rm.persistLocked(run)
}

func (rm *RunManager) onTreeNode(runID string, node crawler.TreeNode) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return
	}
	if _, exists := run.treeNodes[node.ID]; exists {
		return
	}
	run.treeNodes[node.ID] = node
	rm.broadcastLocked(run, crawler.EventFrame{
		Ts:        time.Now().UTC(),
		Status:    run.Status,
		Summary:   rm.summaryLocked(run),
		TreeNodes: []crawler.TreeNode{node},
	})
	_ = rm.persistLocked(run)
}

func (rm *RunManager) onTreeEdge(runID string, edge crawler.TreeEdge) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return
	}
	run.treeEdges = append(run.treeEdges, edge)
	rm.broadcastLocked(run, crawler.EventFrame{
		Ts:        time.Now().UTC(),
		Status:    run.Status,
		Summary:   rm.summaryLocked(run),
		TreeEdges: []crawler.TreeEdge{edge},
	})
	_ = rm.persistLocked(run)
}

func (rm *RunManager) onDiagnostic(runID, kind string, entry crawler.DiagnosticEntry) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return
	}
	switch kind {
	case "skip":
		run.diagnostics.SkippedURLs = append(run.diagnostics.SkippedURLs, entry)
	case "retry":
		run.diagnostics.RetryEvents = append(run.diagnostics.RetryEvents, entry)
	case "error":
		run.diagnostics.Errors = append(run.diagnostics.Errors, entry)
	case "fetch":
		run.diagnostics.FetchLog = append(run.diagnostics.FetchLog, entry)
	}
	_ = rm.persistLocked(run)
}

func (rm *RunManager) onComplete(runID, reason string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	run, ok := rm.runs[runID]
	if !ok {
		return
	}
	now := time.Now().UTC()
	run.Status = "stopped"
	run.StopReason = reason
	run.StoppedAt = &now
	rm.broadcastLocked(run, crawler.EventFrame{
		Ts:      now,
		Status:  run.Status,
		Summary: rm.summaryLocked(run),
	})
	_ = rm.persistLocked(run)
}

func (rm *RunManager) pagesLocked(run *RunState) []crawler.Page {
	out := make([]crawler.Page, 0, len(run.pageOrder))
	for _, id := range run.pageOrder {
		out = append(out, run.pagesByID[id])
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DiscoveredAt.Before(out[j].DiscoveredAt)
	})
	return out
}

func (rm *RunManager) treeNodesLocked(run *RunState) []crawler.TreeNode {
	out := make([]crawler.TreeNode, 0, len(run.treeNodes))
	for _, node := range run.treeNodes {
		page := run.pagesByID[node.ID]
		if page.Title != "" {
			node.Title = page.Title
		}
		out = append(out, node)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Depth == out[j].Depth {
			return out[i].URL < out[j].URL
		}
		return out[i].Depth < out[j].Depth
	})
	return out
}

func (rm *RunManager) summaryLocked(run *RunState) crawler.Summary {
	summary := crawler.Summary{
		Status:      run.Status,
		PagesQueued: len(run.treeNodes),
	}
	for _, page := range run.pagesByID {
		if page.ErrorClass != "" {
			summary.PagesFailed++
		} else if page.StatusCode > 0 {
			summary.PagesFetched++
		}
		if page.StatusCode > 0 {
			fetchedAt := page.FetchedAt
			if summary.LastFetchedAt == nil || fetchedAt.After(*summary.LastFetchedAt) {
				ts := fetchedAt
				summary.LastFetchedAt = &ts
			}
		}
	}
	return summary
}

func (rm *RunManager) snapshotLocked(run *RunState) crawler.Snapshot {
	var rootPageID string
	for _, id := range run.pageOrder {
		if run.pagesByID[id].Depth == 0 {
			rootPageID = id
			break
		}
	}
	return crawler.Snapshot{
		RunID:       run.ID,
		RootPageID:  rootPageID,
		Config:      run.Config,
		Seed:        run.Seed,
		Status:      run.Status,
		StopReason:  run.StopReason,
		CreatedAt:   run.CreatedAt,
		StartedAt:   run.StartedAt,
		StoppedAt:   run.StoppedAt,
		Summary:     rm.summaryLocked(run),
		Pages:       rm.pagesLocked(run),
		TreeNodes:   rm.treeNodesLocked(run),
		TreeEdges:   append([]crawler.TreeEdge(nil), run.treeEdges...),
		Diagnostics: run.diagnostics,
		Paths:       rm.artifacts.Paths(run.ID),
	}
}

func (rm *RunManager) persistLocked(run *RunState) error {
	snapshot := rm.snapshotLocked(run)
	return rm.artifacts.Persist(snapshot)
}

func (rm *RunManager) broadcastLocked(run *RunState, frame crawler.EventFrame) {
	for _, ch := range run.subscribers {
		select {
		case ch <- frame:
		default:
		}
	}
}

func (rm *RunManager) resolveSeed(ctx context.Context, cfg crawler.RunConfig) (crawler.SearchSeed, []crawler.DiagnosticEntry, error) {
	switch cfg.Mode {
	case crawler.SourceModeKeyword:
		resultSet, err := rm.resolver.Resolve(ctx, cfg.Input)
		searchLog := make([]crawler.DiagnosticEntry, 0, len(resultSet.Attempts))
		for _, attempt := range resultSet.Attempts {
			detail := strings.Join(compactNonEmpty([]string{
				attempt.Status,
				attempt.Source,
				fmt.Sprintf("results=%d", attempt.ResultCount),
				fmt.Sprintf("response_bytes=%d", attempt.ResponseBytes),
				attempt.Error,
				attempt.Snippet,
			}), " | ")
			searchLog = append(searchLog, crawler.DiagnosticEntry{
				At:     time.Now().UTC(),
				Reason: "search_attempt",
				Detail: detail,
			})
		}
		if err != nil {
			return crawler.SearchSeed{}, searchLog, err
		}
		results := resultSet.URLs
		if len(results) == 0 {
			return crawler.SearchSeed{}, searchLog, fmt.Errorf("no search results found")
		}
		primary, _, err := crawler.Canonicalize(results[0])
		if err != nil {
			return crawler.SearchSeed{}, searchLog, err
		}
		normalized := make([]string, 0, len(results))
		for _, result := range results {
			if canonical, _, err := crawler.Canonicalize(result); err == nil {
				normalized = append(normalized, canonical)
			}
		}
		return crawler.SearchSeed{
			Query:      cfg.Input,
			PrimaryURL: primary,
			Results:    normalized,
		}, searchLog, nil
	default:
		canonical, _, err := crawler.Canonicalize(cfg.Input)
		if err != nil {
			return crawler.SearchSeed{}, nil, err
		}
		return crawler.SearchSeed{
			Query:      cfg.Input,
			PrimaryURL: canonical,
			Results:    []string{canonical},
		}, nil, nil
	}
}

func (rm *RunManager) applyDefaults(cfg crawler.RunConfig) crawler.RunConfig {
	if cfg.MaxDepth == 0 {
		cfg.MaxDepth = rm.defaults.MaxDepth
	}
	if cfg.MaxPages == 0 {
		cfg.MaxPages = rm.defaults.MaxPages
	}
	if cfg.TimeBudget == 0 && cfg.TimeBudgetSeconds == 0 {
		cfg.TimeBudget = rm.defaults.TimeBudget
	}
	if cfg.MaxLinksPerPage == 0 {
		cfg.MaxLinksPerPage = rm.defaults.MaxLinksPerPage
	}
	if cfg.GlobalConcurrency == 0 {
		cfg.GlobalConcurrency = rm.defaults.GlobalConcurrency
	}
	if cfg.PerHostConcurrency == 0 {
		cfg.PerHostConcurrency = rm.defaults.PerHostConcurrency
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = rm.defaults.UserAgent
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = rm.defaults.RequestTimeout
	}
	if cfg.HeaderTimeout == 0 {
		cfg.HeaderTimeout = rm.defaults.HeaderTimeout
	}
	if cfg.TLSHandshakeTimeout == 0 {
		cfg.TLSHandshakeTimeout = rm.defaults.TLSHandshakeTimeout
	}
	if cfg.IdleConnTimeout == 0 {
		cfg.IdleConnTimeout = rm.defaults.IdleConnTimeout
	}
	if cfg.MaxBodyBytes == 0 {
		cfg.MaxBodyBytes = rm.defaults.MaxBodyBytes
	}
	if cfg.RobotsTTL == 0 {
		cfg.RobotsTTL = rm.defaults.RobotsTTL
	}
	if cfg.RetryBaseDelay == 0 {
		cfg.RetryBaseDelay = rm.defaults.RetryBaseDelay
	}
	if cfg.RetryMax == 0 {
		cfg.RetryMax = rm.defaults.RetryMax
	}
	if cfg.CircuitTripCount == 0 {
		cfg.CircuitTripCount = rm.defaults.CircuitTripCount
	}
	if cfg.CircuitResetTime == 0 {
		cfg.CircuitResetTime = rm.defaults.CircuitResetTime
	}
	cfg.TimeBudget = cfg.Normalize().TimeBudget
	if cfg.Mode == "" {
		cfg.Mode = crawler.SourceModeURL
	}
	return cfg
}

func (rm *RunManager) cloneRun(run *RunState) *RunState {
	copied := *run
	return &copied
}

func formatSearchLog(entries []crawler.DiagnosticEntry) string {
	if len(entries) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, entry.Detail)
	}
	return strings.Join(parts, " || ")
}

func compactNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if strings.TrimSpace(item) == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

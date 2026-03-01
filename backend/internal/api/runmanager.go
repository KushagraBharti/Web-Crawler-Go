package api

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"webcrawler/internal/config"
	"webcrawler/internal/crawler"
	"webcrawler/internal/metrics"
	"webcrawler/internal/storage"
)

type RunState struct {
	ID         uuid.UUID
	Config     crawler.RunConfig
	Status     string
	Engine     *crawler.Engine
	Telemetry  *metrics.Telemetry
	CreatedAt  time.Time
	StartedAt  *time.Time
	StoppedAt  *time.Time
	StopReason string
}

type RunManager struct {
	store    storage.Store
	defaults config.CrawlerDefaults
	mu       sync.RWMutex
	runs     map[uuid.UUID]*RunState
}

func NewRunManager(store storage.Store, defaults config.CrawlerDefaults) *RunManager {
	return &RunManager{
		store:    store,
		defaults: defaults,
		runs:     make(map[uuid.UUID]*RunState),
	}
}

func (rm *RunManager) CreateRun(ctx context.Context, cfg crawler.RunConfig) (uuid.UUID, error) {
	cfg = rm.applyDefaults(cfg)
	cfg = cfg.Normalize()
	id, err := rm.store.CreateRun(ctx, storage.RunConfig{
		SeedURL:            cfg.SeedURL,
		MaxDepth:           cfg.MaxDepth,
		MaxPages:           cfg.MaxPages,
		TimeBudgetSeconds:  int(cfg.TimeBudget.Seconds()),
		MaxLinksPerPage:    cfg.MaxLinksPerPage,
		GlobalConcurrency:  cfg.GlobalConcurrency,
		PerHostConcurrency: cfg.PerHostConcurrency,
		UserAgent:          cfg.UserAgent,
		RespectRobots:      cfg.RespectRobots,
	})
	if err != nil {
		return uuid.Nil, err
	}
	rm.mu.Lock()
	rm.runs[id] = &RunState{ID: id, Config: cfg, Status: "created", CreatedAt: time.Now()}
	rm.mu.Unlock()
	return id, nil
}

func (rm *RunManager) StartRun(ctx context.Context, id uuid.UUID) error {
	rm.mu.RLock()
	state, ok := rm.runs[id]
	if !ok {
		rm.mu.RUnlock()
		return errors.New("run not found")
	}
	if state.Engine != nil {
		rm.mu.RUnlock()
		return errors.New("run already started")
	}
	cfg := state.Config
	rm.mu.RUnlock()

	telemetry := metrics.NewTelemetry()
	engine := crawler.NewEngine(id, cfg, rm.store, telemetry)
	now := time.Now()

	if err := rm.store.UpdateRunStatus(ctx, id, "running", &now, nil, nil); err != nil {
		return err
	}

	rm.mu.Lock()
	state, ok = rm.runs[id]
	if !ok {
		rm.mu.Unlock()
		engine.StopWithReason(crawler.StopReasonUnknown)
		return errors.New("run not found")
	}
	if state.Engine != nil {
		rm.mu.Unlock()
		engine.StopWithReason(crawler.StopReasonUnknown)
		return errors.New("run already started")
	}
	state.Engine = engine
	state.Telemetry = telemetry
	state.Status = "running"
	state.StartedAt = &now
	rm.mu.Unlock()

	engine.Start(cfg.SeedURL)
	go func() {
		<-engine.Done()
		stoppedAt := time.Now()
		stopReason := engine.StopReason()
		if stopReason == "" {
			stopReason = crawler.StopReasonUnknown
		}
		rm.mu.Lock()
		if state, ok := rm.runs[id]; ok {
			state.Status = "stopped"
			state.StoppedAt = &stoppedAt
			state.StopReason = stopReason
		}
		rm.mu.Unlock()
	}()
	return nil
}

func (rm *RunManager) StopRun(ctx context.Context, id uuid.UUID) error {
	rm.mu.RLock()
	state, ok := rm.runs[id]
	if !ok {
		rm.mu.RUnlock()
		return errors.New("run not found")
	}
	engine := state.Engine
	rm.mu.RUnlock()
	if engine != nil {
		engine.StopWithReason(crawler.StopReasonManual)
	}
	now := time.Now()
	stopReason := crawler.StopReasonManual
	if err := rm.store.UpdateRunStatus(ctx, id, "stopped", nil, &now, &stopReason); err != nil {
		return err
	}
	rm.mu.Lock()
	if state, ok := rm.runs[id]; ok {
		state.Status = "stopped"
		state.StoppedAt = &now
		state.StopReason = stopReason
	}
	rm.mu.Unlock()
	return nil
}

func (rm *RunManager) GetRun(ctx context.Context, id uuid.UUID) (RunState, error) {
	rm.mu.RLock()
	state, ok := rm.runs[id]
	if ok {
		snapshot := cloneRunState(state)
		rm.mu.RUnlock()
		return snapshot, nil
	}
	rm.mu.RUnlock()
	row, err := rm.store.GetRun(ctx, id)
	if err != nil {
		return RunState{}, err
	}
	cfg := crawler.RunConfig{
		SeedURL:            row.SeedURL,
		MaxDepth:           row.MaxDepth,
		MaxPages:           row.MaxPages,
		TimeBudgetSeconds:  row.TimeBudgetSeconds,
		MaxLinksPerPage:    row.MaxLinksPerPage,
		GlobalConcurrency:  row.GlobalConcurrency,
		PerHostConcurrency: row.PerHostConcurrency,
		UserAgent:          row.UserAgent,
		RespectRobots:      row.RespectRobots,
	}
	return RunState{
		ID:        row.ID,
		Config:    cfg.Normalize(),
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
		StartedAt: func() *time.Time {
			if row.StartedAt.Valid {
				t := row.StartedAt.Time
				return &t
			}
			return nil
		}(),
		StoppedAt: func() *time.Time {
			if row.StoppedAt.Valid {
				t := row.StoppedAt.Time
				return &t
			}
			return nil
		}(),
		StopReason: func() string {
			if row.StopReason.Valid {
				return row.StopReason.String
			}
			return ""
		}(),
	}, nil
}

func (rm *RunManager) TelemetryFor(id uuid.UUID) (*metrics.Telemetry, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	state, ok := rm.runs[id]
	if !ok || state.Telemetry == nil {
		return nil, false
	}
	return state.Telemetry, true
}

func (rm *RunManager) Summary(ctx context.Context, id uuid.UUID) (storage.RunSummary, error) {
	return rm.store.GetRunSummary(ctx, id)
}

func (rm *RunManager) ListPages(ctx context.Context, id uuid.UUID, limit int) ([]storage.PageRow, error) {
	return rm.store.ListPages(ctx, id, limit)
}

func (rm *RunManager) StartRetentionCleaner(ctx context.Context, interval, retention time.Duration) {
	if retention <= 0 {
		return
	}
	if interval <= 0 {
		interval = time.Hour
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rm.pruneOnce(ctx, retention)
			}
		}
	}()
}

func (rm *RunManager) pruneOnce(ctx context.Context, retention time.Duration) {
	before := time.Now().Add(-retention)
	exclude := rm.activeRunIDs()
	deleted, err := rm.store.PruneRunsOlderThan(ctx, before, exclude)
	if err != nil {
		log.Printf("retention prune failed: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("retention prune deleted %d runs older than %s", deleted, before.UTC().Format(time.RFC3339))
	}
}

func (rm *RunManager) activeRunIDs() []uuid.UUID {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	out := make([]uuid.UUID, 0, len(rm.runs))
	for id, state := range rm.runs {
		if state == nil {
			continue
		}
		if state.Status == "running" || (state.Engine != nil && state.StopReason == "") {
			out = append(out, id)
		}
	}
	return out
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
	return cfg
}

func cloneRunState(state *RunState) RunState {
	snapshot := *state
	if state.StartedAt != nil {
		t := *state.StartedAt
		snapshot.StartedAt = &t
	}
	if state.StoppedAt != nil {
		t := *state.StoppedAt
		snapshot.StoppedAt = &t
	}
	return snapshot
}

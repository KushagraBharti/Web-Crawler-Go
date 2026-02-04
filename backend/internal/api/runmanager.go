package api

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"webcrawler/internal/config"
	"webcrawler/internal/crawler"
	"webcrawler/internal/metrics"
	"webcrawler/internal/storage"
)

type RunState struct {
	ID        uuid.UUID
	Config    crawler.RunConfig
	Status    string
	Engine    *crawler.Engine
	Telemetry *metrics.Telemetry
	CreatedAt time.Time
	StartedAt *time.Time
	StoppedAt *time.Time
}

type RunManager struct {
	store    storage.Store
	defaults config.CrawlerDefaults
	mu       sync.Mutex
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
	rm.mu.Lock()
	state, ok := rm.runs[id]
	rm.mu.Unlock()
	if !ok {
		return errors.New("run not found")
	}
	if state.Engine != nil {
		return errors.New("run already started")
	}

	telemetry := metrics.NewTelemetry()
	engine := crawler.NewEngine(id, state.Config, rm.store, telemetry)
	now := time.Now()
	state.Engine = engine
	state.Telemetry = telemetry
	state.Status = "running"
	state.StartedAt = &now

	if err := rm.store.UpdateRunStatus(ctx, id, "running", &now, nil); err != nil {
		return err
	}
	engine.Start(state.Config.SeedURL)
	go func() {
		<-engine.Done()
		rm.mu.Lock()
		defer rm.mu.Unlock()
		stoppedAt := time.Now()
		state.Status = "stopped"
		state.StoppedAt = &stoppedAt
	}()
	return nil
}

func (rm *RunManager) StopRun(ctx context.Context, id uuid.UUID) error {
	rm.mu.Lock()
	state, ok := rm.runs[id]
	rm.mu.Unlock()
	if !ok {
		return errors.New("run not found")
	}
	if state.Engine != nil {
		state.Engine.Stop()
	}
	now := time.Now()
	state.Status = "stopped"
	state.StoppedAt = &now
	return rm.store.UpdateRunStatus(ctx, id, "stopped", nil, &now)
}

func (rm *RunManager) GetRun(ctx context.Context, id uuid.UUID) (RunState, error) {
	rm.mu.Lock()
	state, ok := rm.runs[id]
	rm.mu.Unlock()
	if ok {
		return *state, nil
	}
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
	}, nil
}

func (rm *RunManager) TelemetryFor(id uuid.UUID) (*metrics.Telemetry, bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	state, ok := rm.runs[id]
	if !ok || state.Telemetry == nil {
		return nil, false
	}
	return state.Telemetry, true
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

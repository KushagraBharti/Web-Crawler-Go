package storage

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu    sync.Mutex
	runs  map[uuid.UUID]RunRow
	pages []PageRecord
	edges map[string]int
	errors []struct {
		runID   uuid.UUID
		host    string
		url     string
		class   string
		message string
		at      time.Time
	}
}

func NewMemory() *MemoryStore {
	return &MemoryStore{
		runs:  make(map[uuid.UUID]RunRow),
		edges: make(map[string]int),
	}
}

func (m *MemoryStore) Migrate(ctx context.Context) error {
	return nil
}

func (m *MemoryStore) CreateRun(ctx context.Context, cfg RunConfig) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	m.runs[id] = RunRow{
		ID:                 id,
		SeedURL:            cfg.SeedURL,
		Status:             "created",
		CreatedAt:          time.Now(),
		MaxDepth:           cfg.MaxDepth,
		MaxPages:           cfg.MaxPages,
		TimeBudgetSeconds:  cfg.TimeBudgetSeconds,
		MaxLinksPerPage:    cfg.MaxLinksPerPage,
		GlobalConcurrency:  cfg.GlobalConcurrency,
		PerHostConcurrency: cfg.PerHostConcurrency,
		UserAgent:          cfg.UserAgent,
		RespectRobots:      cfg.RespectRobots,
	}
	return id, nil
}

func (m *MemoryStore) UpdateRunStatus(ctx context.Context, id uuid.UUID, status string, startedAt, stoppedAt *time.Time, stopReason *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[id]
	if !ok {
		return errors.New("run not found")
	}
	run.Status = status
	if startedAt != nil {
		run.StartedAt = sqlNullTime(*startedAt)
	}
	if stoppedAt != nil {
		run.StoppedAt = sqlNullTime(*stoppedAt)
	}
	if stopReason != nil {
		run.StopReason = sql.NullString{String: *stopReason, Valid: *stopReason != ""}
	}
	m.runs[id] = run
	return nil
}

func (m *MemoryStore) GetRun(ctx context.Context, id uuid.UUID) (RunRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.runs[id]
	if !ok {
		return RunRow{}, errors.New("run not found")
	}
	return run, nil
}

func (m *MemoryStore) GetRunSummary(ctx context.Context, id uuid.UUID) (RunSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var summary RunSummary
	hostSet := make(map[string]struct{})
	var lastFetched *time.Time
	for _, page := range m.pages {
		if page.RunID != id {
			continue
		}
		if page.ErrClass == "" {
			summary.PagesFetched++
		} else {
			summary.PagesFailed++
		}
		if page.Host != "" {
			hostSet[page.Host] = struct{}{}
		}
		summary.TotalBytes += page.SizeBytes
		if page.FetchedAt != nil {
			if lastFetched == nil || page.FetchedAt.After(*lastFetched) {
				lastFetched = page.FetchedAt
			}
		}
	}
	summary.UniqueHosts = int64(len(hostSet))
	summary.LastFetchedAt = lastFetched
	return summary, nil
}

func (m *MemoryStore) ListPages(ctx context.Context, id uuid.UUID, limit int) ([]PageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	var rows []PageRow
	for _, page := range m.pages {
		if page.RunID != id {
			continue
		}
		rows = append(rows, PageRow{
			URL:          page.URL,
			Host:         page.Host,
			Depth:        page.Depth,
			StatusCode:   page.StatusCode,
			ContentType:  page.ContentType,
			FetchMS:      page.FetchMS,
			SizeBytes:    page.SizeBytes,
			ErrorClass:   page.ErrClass,
			ErrorMessage: page.ErrMessage,
			FetchedAt:    page.FetchedAt,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		ti := rows[i].FetchedAt
		tj := rows[j].FetchedAt
		if ti == nil && tj == nil {
			return rows[i].URL > rows[j].URL
		}
		if ti == nil {
			return false
		}
		if tj == nil {
			return true
		}
		return ti.After(*tj)
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows, nil
}

func (m *MemoryStore) InsertPage(ctx context.Context, rec PageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pages = append(m.pages, rec)
	return nil
}

func (m *MemoryStore) InsertError(ctx context.Context, runID uuid.UUID, host, url, class, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors = append(m.errors, struct {
		runID   uuid.UUID
		host    string
		url     string
		class   string
		message string
		at      time.Time
	}{runID: runID, host: host, url: url, class: class, message: message, at: time.Now()})
	return nil
}

func (m *MemoryStore) UpsertEdge(ctx context.Context, runID uuid.UUID, src, dst string, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := runID.String() + ":" + src + "->" + dst
	m.edges[key] += count
	return nil
}

func (m *MemoryStore) UpsertHostStat(ctx context.Context, runID uuid.UUID, host string, bucket time.Time, req, errCount, p50, p95 int, bytes int64, reuse float64) error {
	return nil
}

func sqlNullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}
